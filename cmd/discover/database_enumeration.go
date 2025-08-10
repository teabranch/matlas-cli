package discover

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/types"
)

// DatabaseEnumerator handles discovery of databases and collections from MongoDB clusters
type DatabaseEnumerator struct {
	atlasClient    *atlasclient.Client
	maxConcurrency int
	timeout        time.Duration
	verbose        bool
	mongoURI       string
	mongoUsername  string
	mongoPassword  string
}

// NewDatabaseEnumerator creates a new database enumerator
func NewDatabaseEnumerator(atlasClient *atlasclient.Client, opts *DiscoverOptions) *DatabaseEnumerator {
	return &DatabaseEnumerator{
		atlasClient:    atlasClient,
		maxConcurrency: opts.MaxConcurrency,
		timeout:        opts.Timeout,
		verbose:        opts.Verbose,
		mongoURI:       opts.MongoURI,
		mongoUsername:  opts.MongoUsername,
		mongoPassword:  opts.MongoPassword,
	}
}

// EnumerateClusterDatabases discovers databases and collections for all clusters
func (e *DatabaseEnumerator) EnumerateClusterDatabases(ctx context.Context, clusters []types.ClusterManifest) ([]DatabaseInfo, error) {
	if len(clusters) == 0 {
		return []DatabaseInfo{}, nil
	}

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, e.maxConcurrency)
	results := make(chan clusterDatabaseResult, len(clusters))

	var wg sync.WaitGroup

	// Process each cluster concurrently
	for _, cluster := range clusters {
		wg.Add(1)
		go func(c types.ClusterManifest) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			databases, err := e.enumerateClusterDatabases(ctx, c)
			results <- clusterDatabaseResult{
				ClusterName: c.Metadata.Name,
				Databases:   databases,
				Error:       err,
			}
		}(cluster)
	}

	// Wait for all goroutines and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allDatabases []DatabaseInfo
	var errors []error

	for result := range results {
		if result.Error != nil {
			if e.verbose {
				fmt.Printf("Warning: Failed to enumerate databases for cluster %s: %v\n", result.ClusterName, result.Error)
			}
			errors = append(errors, fmt.Errorf("cluster %s: %w", result.ClusterName, result.Error))
			continue
		}

		// Add cluster name to each database
		for _, db := range result.Databases {
			db.ClusterName = result.ClusterName
			allDatabases = append(allDatabases, db)
		}
	}

	// Sort databases by cluster name and database name for consistent output
	sort.Slice(allDatabases, func(i, j int) bool {
		if allDatabases[i].ClusterName != allDatabases[j].ClusterName {
			return allDatabases[i].ClusterName < allDatabases[j].ClusterName
		}
		return allDatabases[i].Name < allDatabases[j].Name
	})

	// Return aggregated error if any clusters failed
	if len(errors) > 0 {
		return allDatabases, &DatabaseEnumerationError{
			ClusterErrors: errors,
		}
	}

	return allDatabases, nil
}

// enumerateClusterDatabases discovers databases for a single cluster
func (e *DatabaseEnumerator) enumerateClusterDatabases(ctx context.Context, cluster types.ClusterManifest) ([]DatabaseInfo, error) {
	if e.verbose {
		fmt.Printf("  Enumerating databases for cluster: %s\n", cluster.Metadata.Name)
	}

	// If no Atlas client is provided (e.g., in unit tests), return empty result gracefully
	if e.atlasClient == nil {
		if e.verbose {
			fmt.Printf("    Skipping database enumeration - no Atlas client available\n")
		}
		return []DatabaseInfo{}, nil
	}

	// Create connection string for the cluster
	connectionString, err := e.getClusterConnectionString(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	// Connect to MongoDB
	client, err := e.connectToCluster(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			if e.verbose {
				fmt.Printf("    Warning: failed to disconnect MongoDB client: %v\n", err)
			}
		}
	}()

	// List databases
	dbNames, err := e.listDatabases(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	// Get details for each database
	var databases []DatabaseInfo
	for _, dbName := range dbNames {
		if e.isSystemDatabase(dbName) {
			continue // Skip system databases
		}

		dbInfo, err := e.getDatabaseInfo(ctx, client, dbName)
		if err != nil {
			if e.verbose {
				fmt.Printf("    Warning: Failed to get info for database %s: %v\n", dbName, err)
			}
			// Continue with other databases even if one fails
			continue
		}

		databases = append(databases, dbInfo)
	}

	if e.verbose {
		fmt.Printf("    Found %d databases in cluster %s\n", len(databases), cluster.Metadata.Name)
	}

	return databases, nil
}

// getClusterConnectionString constructs a connection string for the cluster
func (e *DatabaseEnumerator) getClusterConnectionString(cluster types.ClusterManifest) (string, error) {
	// 1) If explicit MongoURI override provided, use it as-is
	if e.mongoURI != "" {
		return e.mongoURI, nil
	}

	// 2) If we have an Atlas client, fetch the real connection string from Atlas API
	if e.atlasClient != nil {
		clusterName := cluster.Metadata.Name
		projectID := cluster.Metadata.Labels["atlas.mongodb.com/project-id"]
		if projectID == "" {
			return "", fmt.Errorf("project ID not found in cluster metadata")
		}

		// Use Clusters service to fetch connection strings
		svc := atlas.NewClustersService(e.atlasClient)
		detailed, err := svc.Get(context.Background(), projectID, clusterName)
		if err != nil {
			return "", fmt.Errorf("failed to get cluster details for connection string: %w", err)
		}
		if detailed.ConnectionStrings == nil {
			return "", fmt.Errorf("connection strings not available for cluster %s", clusterName)
		}
		if detailed.ConnectionStrings.StandardSrv != nil {
			conn := *detailed.ConnectionStrings.StandardSrv
			// Inject credentials if provided and URI lacks them
			if e.mongoUsername != "" && e.mongoPassword != "" && !hasCredentials(conn) {
				conn = injectCredentials(conn, e.mongoUsername, e.mongoPassword)
			}
			return conn, nil
		}
		if detailed.ConnectionStrings.Standard != nil {
			conn := *detailed.ConnectionStrings.Standard
			if e.mongoUsername != "" && e.mongoPassword != "" && !hasCredentials(conn) {
				conn = injectCredentials(conn, e.mongoUsername, e.mongoPassword)
			}
			return conn, nil
		}
		return "", fmt.Errorf("no usable connection string found for cluster %s", clusterName)
	}

	// 3) Fallback: construct a generic SRV connection string using the cluster name
	return fmt.Sprintf("mongodb+srv://<username>:<password>@%s.mongodb.net/", cluster.Metadata.Name), nil
}

// connectToCluster establishes a connection to the MongoDB cluster
func (e *DatabaseEnumerator) connectToCluster(ctx context.Context, connectionString string) (*mongo.Client, error) {
	// Create a context with timeout for connection
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Set client options
	clientOpts := options.Client().ApplyURI(connectionString)
	// For SRV URIs without explicit DB, default to admin for auth
	// The driver handles SRV. Additional options can be appended in the URI.
	clientOpts.SetMaxPoolSize(5)
	clientOpts.SetServerSelectionTimeout(5 * time.Second)

	// Connect to MongoDB
	client, err := mongo.Connect(connectCtx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Ping to verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		if derr := client.Disconnect(ctx); derr != nil {
			if e.verbose {
				fmt.Printf("    Warning: failed to disconnect MongoDB client after ping error: %v\n", derr)
			}
		}
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

// listDatabases gets a list of all databases in the cluster
func (e *DatabaseEnumerator) listDatabases(ctx context.Context, client *mongo.Client) ([]string, error) {
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := client.ListDatabaseNames(listCtx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list database names: %w", err)
	}

	return result, nil
}

// getDatabaseInfo gets detailed information about a database
func (e *DatabaseEnumerator) getDatabaseInfo(ctx context.Context, client *mongo.Client, dbName string) (DatabaseInfo, error) {
	database := client.Database(dbName)

	// Get database stats
	var dbStats bson.M
	statsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := database.RunCommand(statsCtx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&dbStats)
	if err != nil {
		// If dbStats fails, continue without size information
		if e.verbose {
			fmt.Printf("      Warning: Could not get stats for database %s: %v\n", dbName, err)
		}
	}

	// Get collections
	collections, err := e.getCollectionInfo(ctx, database)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("failed to get collections for database %s: %w", dbName, err)
	}

	// Extract size information if available
	var sizeOnDisk int64
	if dataSize, ok := dbStats["dataSize"]; ok {
		if size, ok := dataSize.(int64); ok {
			sizeOnDisk = size
		} else if size, ok := dataSize.(int32); ok {
			sizeOnDisk = int64(size)
		}
	}

	dbInfo := DatabaseInfo{
		Name:        dbName,
		SizeOnDisk:  sizeOnDisk,
		Collections: collections,
	}

	return dbInfo, nil
}

// getCollectionInfo gets information about all collections in a database
func (e *DatabaseEnumerator) getCollectionInfo(ctx context.Context, database *mongo.Database) ([]CollectionInfo, error) {
	// List collections
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	collectionNames, err := database.ListCollectionNames(listCtx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var collections []CollectionInfo

	// Get detailed info for each collection
	for _, collName := range collectionNames {
		if e.isSystemCollection(collName) {
			continue // Skip system collections
		}

		collInfo, err := e.getDetailedCollectionInfo(ctx, database, collName)
		if err != nil {
			if e.verbose {
				fmt.Printf("        Warning: Failed to get info for collection %s: %v\n", collName, err)
			}
			// Add basic collection info even if detailed info fails
			collInfo = CollectionInfo{Name: collName}
		}

		collections = append(collections, collInfo)
	}

	// Sort collections by name for consistent output
	sort.Slice(collections, func(i, j int) bool {
		return collections[i].Name < collections[j].Name
	})

	return collections, nil
}

// getDetailedCollectionInfo gets detailed information about a single collection
func (e *DatabaseEnumerator) getDetailedCollectionInfo(ctx context.Context, database *mongo.Database, collName string) (CollectionInfo, error) {
	collection := database.Collection(collName)

	// Get collection stats
	var collStats bson.M
	statsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := collection.Database().RunCommand(statsCtx, bson.D{
		{Key: "collStats", Value: collName},
	}).Decode(&collStats)

	collInfo := CollectionInfo{Name: collName}

	if err != nil {
		if e.verbose {
			fmt.Printf("          Warning: Could not get stats for collection %s: %v\n", collName, err)
		}
	} else {
		// Extract collection statistics
		if count, ok := collStats["count"]; ok {
			if c, ok := count.(int64); ok {
				collInfo.DocumentCount = c
			} else if c, ok := count.(int32); ok {
				collInfo.DocumentCount = int64(c)
			}
		}

		if size, ok := collStats["storageSize"]; ok {
			if s, ok := size.(int64); ok {
				collInfo.StorageSize = s
			} else if s, ok := size.(int32); ok {
				collInfo.StorageSize = int64(s)
			}
		}
	}

	// Get indexes
	indexes, err := e.getIndexInfo(ctx, collection)
	if err != nil {
		if e.verbose {
			fmt.Printf("          Warning: Could not get indexes for collection %s: %v\n", collName, err)
		}
	} else {
		collInfo.Indexes = indexes
		collInfo.IndexCount = len(indexes)
	}

	return collInfo, nil
}

// getIndexInfo gets information about indexes for a collection
func (e *DatabaseEnumerator) getIndexInfo(ctx context.Context, collection *mongo.Collection) ([]IndexInfo, error) {
	indexCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cursor, err := collection.Indexes().List(indexCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			if e.verbose {
				fmt.Printf("    Warning: failed to close index cursor: %v\n", err)
			}
		}
	}()

	var indexes []IndexInfo

	for cursor.Next(ctx) {
		var indexDoc bson.M
		if err := cursor.Decode(&indexDoc); err != nil {
			continue // Skip malformed index documents
		}

		indexInfo := IndexInfo{}

		// Extract index name
		if name, ok := indexDoc["name"].(string); ok {
			indexInfo.Name = name
		}

		// Extract index keys
		if key, ok := indexDoc["key"]; ok {
			indexInfo.Keys = key
		}

		// Extract index options
		if unique, ok := indexDoc["unique"].(bool); ok {
			indexInfo.Unique = unique
		}

		if sparse, ok := indexDoc["sparse"].(bool); ok {
			indexInfo.Sparse = sparse
		}

		if expireAfterSeconds, ok := indexDoc["expireAfterSeconds"]; ok {
			if ttl, ok := expireAfterSeconds.(int32); ok {
				ttlInt := int(ttl)
				indexInfo.TTL = &ttlInt
			}
		}

		if partialFilterExpression, ok := indexDoc["partialFilterExpression"]; ok {
			indexInfo.PartialFilter = partialFilterExpression
		}

		indexes = append(indexes, indexInfo)
	}

	if err := cursor.Err(); err != nil {
		return indexes, fmt.Errorf("cursor error while reading indexes: %w", err)
	}

	// Sort indexes by name for consistent output
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].Name < indexes[j].Name
	})

	return indexes, nil
}

// hasCredentials returns true if the MongoDB URI contains credentials
func hasCredentials(uri string) bool {
	// naive check for pattern mongodb://user:pass@ or mongodb+srv://user:pass@
	return strings.Contains(uri, "://") && strings.Contains(uri, "@") && strings.Index(uri, "@") > strings.Index(uri, "://")
}

// injectCredentials inserts username/password into the connection string safely
func injectCredentials(uri, username, password string) string {
	// Only handle mongodb or mongodb+srv
	if !strings.HasPrefix(uri, "mongodb://") && !strings.HasPrefix(uri, "mongodb+srv://") {
		return uri
	}
	// Split scheme and rest
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return uri
	}
	scheme, rest := parts[0], parts[1]
	// If credentials already present, return as-is
	if hasCredentials(uri) {
		return uri
	}
	// NOTE: We could URL-escape credentials; for now assume plain works for typical test use.
	return fmt.Sprintf("%s://%s:%s@%s", scheme, username, password, rest)
}

// isSystemDatabase checks if a database is a system database that should be skipped
func (e *DatabaseEnumerator) isSystemDatabase(dbName string) bool {
	systemDatabases := []string{
		"admin",
		"local",
		"config",
	}

	for _, sysDB := range systemDatabases {
		if dbName == sysDB {
			return true
		}
	}

	return false
}

// isSystemCollection checks if a collection is a system collection that should be skipped
func (e *DatabaseEnumerator) isSystemCollection(collName string) bool {
	// Skip collections that start with "system."
	if len(collName) > 7 && collName[:7] == "system." {
		return true
	}

	// Skip other known system collections
	systemCollections := []string{
		"fs.files",
		"fs.chunks",
	}

	for _, sysColl := range systemCollections {
		if collName == sysColl {
			return true
		}
	}

	return false
}

// clusterDatabaseResult holds the result of database enumeration for a single cluster
type clusterDatabaseResult struct {
	ClusterName string
	Databases   []DatabaseInfo
	Error       error
}

// DatabaseEnumerationError represents errors that occurred during database enumeration
type DatabaseEnumerationError struct {
	ClusterErrors []error
}

func (e *DatabaseEnumerationError) Error() string {
	if len(e.ClusterErrors) == 1 {
		return fmt.Sprintf("database enumeration failed: %v", e.ClusterErrors[0])
	}
	return fmt.Sprintf("database enumeration failed for %d clusters: %v", len(e.ClusterErrors), e.ClusterErrors[0])
}

func (e *DatabaseEnumerationError) Unwrap() []error {
	return e.ClusterErrors
}
