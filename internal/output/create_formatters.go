package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/teabranch/matlas-cli/internal/config"
)

// CreateResultFormatter provides pretty formatting for create command results
type CreateResultFormatter struct {
	format config.OutputFormat
	writer io.Writer
}

// NewCreateResultFormatter creates a new formatter specifically for create command results
func NewCreateResultFormatter(format config.OutputFormat, writer io.Writer) *CreateResultFormatter {
	return &CreateResultFormatter{
		format: format,
		writer: writer,
	}
}

// FormatCreateResult formats the result of a create operation with prettier output
func (f *CreateResultFormatter) FormatCreateResult(result interface{}, resourceType string) error {
	switch f.format {
	case config.OutputJSON:
		return f.formatJSON(result)
	case config.OutputYAML:
		return f.formatYAML(result)
	case config.OutputTable, config.OutputText, "":
		return f.formatCreateResultText(result, resourceType)
	default:
		return fmt.Errorf("unsupported output format: %s", f.format)
	}
}

// formatCreateResultText formats create results as pretty text output
func (f *CreateResultFormatter) formatCreateResultText(result interface{}, resourceType string) error {
	if result == nil {
		return nil
	}

	// Write success header
	fmt.Fprintf(f.writer, "✅ %s created successfully!\n\n", capitalizeFirst(resourceType))

	// Format based on resource type
	switch resourceType {
	case "project":
		return f.formatProjectCreateResult(result)
	case "cluster":
		return f.formatClusterCreateResult(result)
	case "database user", "user":
		return f.formatUserCreateResult(result)
	case "network access entry", "network":
		return f.formatNetworkCreateResult(result)
	case "network container":
		return f.formatNetworkContainerCreateResult(result)
	case "network peering":
		return f.formatNetworkPeeringCreateResult(result)
	case "vpc endpoint":
		return f.formatVPCEndpointCreateResult(result)
	case "search index":
		return f.formatSearchIndexCreateResult(result)
	case "database":
		return f.formatDatabaseCreateResult(result)
	case "collection":
		return f.formatCollectionCreateResult(result)
	case "index":
		return f.formatIndexCreateResult(result)
	default:
		// Fallback to improved generic formatting
		return f.formatGenericCreateResult(result, resourceType)
	}
}

// formatProjectCreateResult formats project creation results
func (f *CreateResultFormatter) formatProjectCreateResult(result interface{}) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Extract key fields
	if id := getStringField(v, "Id"); id != "" {
		fmt.Fprintf(w, "Project ID:\t%s\n", id)
	}
	if name := getStringField(v, "Name"); name != "" {
		fmt.Fprintf(w, "Name:\t%s\n", name)
	}
	if orgId := getStringField(v, "OrgId"); orgId != "" {
		fmt.Fprintf(w, "Organization ID:\t%s\n", orgId)
	}
	if created := getTimeField(v, "Created"); !created.IsZero() {
		fmt.Fprintf(w, "Created:\t%s\n", created.Format("2006-01-02 15:04:05 UTC"))
	}

	return nil
}

// formatClusterCreateResult formats cluster creation results
func (f *CreateResultFormatter) formatClusterCreateResult(result interface{}) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Extract key fields
	if name := getStringField(v, "Name"); name != "" {
		fmt.Fprintf(w, "Cluster Name:\t%s\n", name)
	}
	if clusterType := getStringField(v, "ClusterType"); clusterType != "" {
		fmt.Fprintf(w, "Type:\t%s\n", clusterType)
	}
	if version := getStringField(v, "MongoDBMajorVersion", "MongoDBVersion"); version != "" {
		fmt.Fprintf(w, "MongoDB Version:\t%s\n", version)
	}
	if stateName := getStringField(v, "StateName"); stateName != "" {
		fmt.Fprintf(w, "Status:\t%s\n", stateName)
	}

	// Show provider and region info from replication specs
	if provider, region := getProviderAndRegion(v); provider != "" {
		fmt.Fprintf(w, "Provider:\t%s\n", provider)
		if region != "" {
			fmt.Fprintf(w, "Region:\t%s\n", region)
		}
	}

	// Show connection info if available (mask credentials if present)
	if connectionStrings := getConnectionStrings(v); len(connectionStrings) > 0 {
		fmt.Fprintf(w, "\nConnection Strings:\n")
		for name, uri := range connectionStrings {
			fmt.Fprintf(w, "  %s:\t%s\n", name, maskConnectionString(uri))
		}
	}

	fmt.Fprintf(w, "\n💡 Tip:\tUse 'matlas atlas clusters get %s' to check deployment progress\n", getStringField(v, "Name"))

	return nil
}

// formatUserCreateResult formats database user creation results
func (f *CreateResultFormatter) formatUserCreateResult(result interface{}) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Extract key fields
	if username := getStringField(v, "Username"); username != "" {
		fmt.Fprintf(w, "Username:\t%s\n", username)
	}
	if dbName := getStringField(v, "DatabaseName"); dbName != "" {
		fmt.Fprintf(w, "Database:\t%s\n", dbName)
	}

	// Format roles nicely
	if roles := getRoles(v); len(roles) > 0 {
		fmt.Fprintf(w, "Roles:\t%s\n", strings.Join(roles, ", "))
	}

	// Show authentication types if not NONE
	authTypes := getAuthenticationTypes(v)
	if len(authTypes) > 0 {
		fmt.Fprintf(w, "Authentication:\t%s\n", strings.Join(authTypes, ", "))
	}

	if groupId := getStringField(v, "GroupId"); groupId != "" {
		fmt.Fprintf(w, "Project ID:\t%s\n", groupId)
	}

	fmt.Fprintf(w, "\n💡 Tip:\tUser password is not shown for security reasons\n")

	return nil
}

// formatNetworkCreateResult formats network access entry creation results
func (f *CreateResultFormatter) formatNetworkCreateResult(result interface{}) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Handle both single entry and slice results
	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// If it's a slice, take the first element
	if v.Kind() == reflect.Slice && v.Len() > 0 {
		v = v.Index(0)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
	}

	// Extract access details
	if ipAddress := getStringField(v, "IpAddress"); ipAddress != "" {
		fmt.Fprintf(w, "IP Address:\t%s\n", ipAddress)
	}
	if cidrBlock := getStringField(v, "CidrBlock"); cidrBlock != "" {
		fmt.Fprintf(w, "CIDR Block:\t%s\n", cidrBlock)
	}
	if awsSG := getStringField(v, "AwsSecurityGroup"); awsSG != "" {
		fmt.Fprintf(w, "AWS Security Group:\t%s\n", awsSG)
	}
	if comment := getStringField(v, "Comment"); comment != "" {
		fmt.Fprintf(w, "Comment:\t%s\n", comment)
	}
	if groupId := getStringField(v, "GroupId"); groupId != "" {
		fmt.Fprintf(w, "Project ID:\t%s\n", groupId)
	}

	fmt.Fprintf(w, "\n💡 Tip:\tChanges may take a few minutes to propagate\n")

	return nil
}

// formatGenericCreateResult provides a fallback formatter for unknown resource types
func (f *CreateResultFormatter) formatGenericCreateResult(result interface{}, resourceType string) error {
	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()

	v := reflect.ValueOf(result)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		fmt.Fprintf(w, "Result:\t%v\n", result)
		return nil
	}

	// Show only the most important/common fields
	priorityFields := []string{"Id", "Name", "Username", "GroupId", "ProjectId", "Status", "State", "StateName"}

	t := v.Type()
	for _, fieldName := range priorityFields {
		if _, found := t.FieldByName(fieldName); found {
			value := v.FieldByName(fieldName)
			if value.IsValid() && !isZeroValue(value) {
				displayName := fieldName
				if fieldName == "GroupId" {
					displayName = "Project ID"
				} else if fieldName == "StateName" {
					displayName = "Status"
				}
				fmt.Fprintf(w, "%s:\t%s\n", displayName, formatFieldValue(value))
			}
		}
	}

	return nil
}

// Helper functions for specific resource types

func (f *CreateResultFormatter) formatNetworkContainerCreateResult(result interface{}) error {
	return f.formatGenericCreateResult(result, "network container")
}

func (f *CreateResultFormatter) formatNetworkPeeringCreateResult(result interface{}) error {
	return f.formatGenericCreateResult(result, "network peering")
}

func (f *CreateResultFormatter) formatVPCEndpointCreateResult(result interface{}) error {
	return f.formatGenericCreateResult(result, "vpc endpoint")
}

func (f *CreateResultFormatter) formatSearchIndexCreateResult(result interface{}) error {
	return f.formatGenericCreateResult(result, "search index")
}

func (f *CreateResultFormatter) formatDatabaseCreateResult(result interface{}) error {
	return f.formatGenericCreateResult(result, "database")
}

func (f *CreateResultFormatter) formatCollectionCreateResult(result interface{}) error {
	return f.formatGenericCreateResult(result, "collection")
}

func (f *CreateResultFormatter) formatIndexCreateResult(result interface{}) error {
	return f.formatGenericCreateResult(result, "index")
}

// Helper functions

func getStringField(v reflect.Value, fieldNames ...string) string {
	// Ensure we're working with a struct, not a pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	for _, fieldName := range fieldNames {
		if field := v.FieldByName(fieldName); field.IsValid() {
			if field.Kind() == reflect.Ptr {
				if !field.IsNil() {
					return field.Elem().String()
				}
			} else if field.Kind() == reflect.String {
				return field.String()
			}
		}
	}
	return ""
}

func getTimeField(v reflect.Value, fieldName string) time.Time {
	// Ensure we're working with a struct, not a pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return time.Time{}
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return time.Time{}
	}

	if field := v.FieldByName(fieldName); field.IsValid() {
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if t, ok := field.Elem().Interface().(time.Time); ok {
				return t
			}
		} else if t, ok := field.Interface().(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

func getProviderAndRegion(v reflect.Value) (string, string) {
	// Ensure we're working with a struct, not a pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", ""
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", ""
	}

	// Try to extract from ReplicationSpecs
	if repSpecs := v.FieldByName("ReplicationSpecs"); repSpecs.IsValid() {
		if repSpecs.Kind() == reflect.Ptr {
			repSpecs = repSpecs.Elem()
		}
		if repSpecs.Kind() == reflect.Slice && repSpecs.Len() > 0 {
			firstSpec := repSpecs.Index(0)
			if firstSpec.Kind() == reflect.Ptr {
				firstSpec = firstSpec.Elem()
			}
			if regionConfigs := firstSpec.FieldByName("RegionConfigs"); regionConfigs.IsValid() {
				if regionConfigs.Kind() == reflect.Ptr {
					regionConfigs = regionConfigs.Elem()
				}
				if regionConfigs.Kind() == reflect.Slice && regionConfigs.Len() > 0 {
					firstRegion := regionConfigs.Index(0)
					if firstRegion.Kind() == reflect.Ptr {
						firstRegion = firstRegion.Elem()
					}
					provider := getStringField(firstRegion, "ProviderName")
					region := getStringField(firstRegion, "RegionName")
					return provider, region
				}
			}
		}
	}
	return "", ""
}

func getConnectionStrings(v reflect.Value) map[string]string {
	result := make(map[string]string)

	// Ensure we're working with a struct, not a pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return result
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return result
	}

	if connStrings := v.FieldByName("ConnectionStrings"); connStrings.IsValid() {
		if connStrings.Kind() == reflect.Ptr {
			if connStrings.IsNil() {
				return result
			}
			connStrings = connStrings.Elem()
		}

		if !connStrings.IsValid() {
			return result
		}

		// Try common connection string field names
		for _, fieldName := range []string{"Standard", "StandardSrv", "Private", "PrivateSrv"} {
			if field := connStrings.FieldByName(fieldName); field.IsValid() {
				if uri := getStringField(field, ""); uri != "" {
					result[fieldName] = uri
				}
			}
		}
	}
	return result
}

// maskConnectionString obfuscates credentials within a MongoDB connection string.
// Examples:
//   - mongodb+srv://user:pass@host/db -> mongodb+srv://user:***@host/db
//   - mongodb://user:p%40ss@host/?x=y -> mongodb://user:***@host/?x=y
//
// If no credentials are present, returns the input unchanged.
func maskConnectionString(uri string) string {
	// Fast path: if there is no '@' there cannot be embedded credentials
	if !strings.Contains(uri, "@") {
		return uri
	}
	// Split scheme and the rest
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return uri
	}
	scheme, rest := parts[0], parts[1]

	// Find credentials segment before '@'
	atIdx := strings.Index(rest, "@")
	if atIdx == -1 {
		return uri
	}

	credsAndHost := rest[:atIdx] // everything before '@'
	// creds may be in the form user or user:password
	colonIdx := strings.Index(credsAndHost, ":")
	if colonIdx == -1 {
		// No password provided, nothing to mask
		return uri
	}
	user := credsAndHost[:colonIdx]
	// Replace password with ***
	maskedCreds := user + ":***"
	masked := scheme + "://" + maskedCreds + rest[atIdx:]
	return masked
}

func getRoles(v reflect.Value) []string {
	var roles []string
	if rolesField := v.FieldByName("Roles"); rolesField.IsValid() {
		if rolesField.Kind() == reflect.Ptr {
			rolesField = rolesField.Elem()
		}
		if rolesField.Kind() == reflect.Slice {
			for i := 0; i < rolesField.Len(); i++ {
				role := rolesField.Index(i)
				if role.Kind() == reflect.Ptr {
					role = role.Elem()
				}

				roleName := getStringField(role, "RoleName")
				databaseName := getStringField(role, "DatabaseName")
				if roleName != "" {
					if databaseName != "" {
						roles = append(roles, fmt.Sprintf("%s@%s", roleName, databaseName))
					} else {
						roles = append(roles, roleName)
					}
				}
			}
		}
	}
	return roles
}

func getAuthenticationTypes(v reflect.Value) []string {
	var authTypes []string

	authFields := map[string]string{
		"AwsIAMType":   "AWS IAM",
		"LdapAuthType": "LDAP",
		"OidcAuthType": "OIDC",
		"X509Type":     "X.509",
	}

	for field, display := range authFields {
		if value := getStringField(v, field); value != "" && value != "NONE" {
			authTypes = append(authTypes, display)
		}
	}

	if len(authTypes) == 0 {
		authTypes = append(authTypes, "Password")
	}

	return authTypes
}

func formatFieldValue(v reflect.Value) string {
	if !v.IsValid() {
		return "<invalid>"
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		return formatFieldValue(v.Elem())
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", v.Float())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	if v.Kind() == reflect.Ptr {
		return v.IsNil()
	}
	zero := reflect.Zero(v.Type())
	return reflect.DeepEqual(v.Interface(), zero.Interface())
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// formatJSON and formatYAML are the same as in the base formatter
func (f *CreateResultFormatter) formatJSON(data interface{}) error {
	formatter := NewFormatter(config.OutputJSON, f.writer)
	return formatter.Format(data)
}

func (f *CreateResultFormatter) formatYAML(data interface{}) error {
	formatter := NewFormatter(config.OutputYAML, f.writer)
	return formatter.Format(data)
}
