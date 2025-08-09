package apply

import (
	"fmt"
	"strings"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// DiscoveredProjectConverter converts DiscoveredProject format to individual resource manifests
type DiscoveredProjectConverter struct{}

// NewDiscoveredProjectConverter creates a new format converter
func NewDiscoveredProjectConverter() *DiscoveredProjectConverter {
	return &DiscoveredProjectConverter{}
}

// ConvertToApplyDocument converts a DiscoveredProject to an ApplyDocument with individual resources
func (c *DiscoveredProjectConverter) ConvertToApplyDocument(discovered interface{}) (*types.ApplyDocument, error) {
	// Parse the discovered data structure
	discoveredMap, ok := discovered.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid discovered project format")
	}

	// Check if it's a DiscoveredProject
	kind, ok := discoveredMap["kind"].(string)
	if !ok || kind != "DiscoveredProject" {
		return nil, fmt.Errorf("not a DiscoveredProject format, got kind: %s", kind)
	}

	// Extract metadata
	metadataRaw, ok := discoveredMap["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid metadata in DiscoveredProject")
	}

	projectID, _ := metadataRaw["projectId"].(string)
	if projectID == "" {
		return nil, fmt.Errorf("missing projectId in DiscoveredProject metadata")
	}

	// Create ApplyDocument
	applyDoc := &types.ApplyDocument{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindApplyDocument,
		Metadata: types.MetadataConfig{
			Name: fmt.Sprintf("converted-project-%s", projectID),
			Labels: map[string]string{
				"converted-from":                "DiscoveredProject",
				"matlas-mongodb-com-project-id": projectID, // Valid label key format
				"conversion-timestamp":          time.Now().UTC().Format("2006-01-02T15-04-05Z"),
			},
			Annotations: map[string]string{
				"description":                   "Auto-converted from DiscoveredProject format",
				"original-kind":                 "DiscoveredProject",
				"matlas.mongodb.com/project-id": projectID, // Store in annotations for reference
			},
		},
		Resources: []types.ResourceManifest{},
	}

	// Extract project name for setting projectName in all resource specs
	var projectName string
	if projectRaw, exists := discoveredMap["project"]; exists && projectRaw != nil {
		if projectMap, ok := projectRaw.(map[string]interface{}); ok {
			if specRaw, exists := projectMap["spec"]; exists {
				if specMap, ok := specRaw.(map[string]interface{}); ok {
					if name, ok := specMap["name"].(string); ok {
						projectName = name
					}
				}
			}
		}
	}

	// Fallback to metadata projectId if no project name found
	if projectName == "" {
		projectName = projectID
	}

	// Convert project if present
	if projectRaw, exists := discoveredMap["project"]; exists && projectRaw != nil {
		if projectManifest, err := c.convertProjectManifest(projectRaw, projectName); err == nil {
			applyDoc.Resources = append(applyDoc.Resources, *projectManifest)
		}
	}

	// Convert clusters
	if clustersRaw, exists := discoveredMap["clusters"]; exists {
		if clusters, ok := clustersRaw.([]interface{}); ok {
			for _, clusterRaw := range clusters {
				if clusterManifest, err := c.convertClusterManifest(clusterRaw, projectName); err == nil {
					applyDoc.Resources = append(applyDoc.Resources, *clusterManifest)
				}
			}
		}
	}

	// Convert database users
	if usersRaw, exists := discoveredMap["databaseUsers"]; exists {
		if users, ok := usersRaw.([]interface{}); ok {
			for _, userRaw := range users {
				if userManifest, err := c.convertDatabaseUserManifest(userRaw, projectName); err == nil {
					applyDoc.Resources = append(applyDoc.Resources, *userManifest)
				}
			}
		}
	}

	// Convert network access
	if networkRaw, exists := discoveredMap["networkAccess"]; exists {
		if networkEntries, ok := networkRaw.([]interface{}); ok {
			for _, entryRaw := range networkEntries {
				if networkManifest, err := c.convertNetworkAccessManifest(entryRaw, projectName); err == nil {
					applyDoc.Resources = append(applyDoc.Resources, *networkManifest)
				}
			}
		}
	}

	return applyDoc, nil
}

// convertProjectManifest converts a project manifest from discovered format
func (c *DiscoveredProjectConverter) convertProjectManifest(projectRaw interface{}, projectName string) (*types.ResourceManifest, error) {
	projectMap, ok := projectRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid project manifest format")
	}

	metadata := c.extractMetadata(projectMap, "project")
	spec := c.extractSpec(projectMap, projectName)

	return &types.ResourceManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindProject,
		Metadata:   metadata,
		Spec:       spec,
	}, nil
}

// convertClusterManifest converts a cluster manifest from discovered format
func (c *DiscoveredProjectConverter) convertClusterManifest(clusterRaw interface{}, projectName string) (*types.ResourceManifest, error) {
	clusterMap, ok := clusterRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid cluster manifest format")
	}

	metadata := c.extractMetadata(clusterMap, "cluster")
	spec := c.extractSpec(clusterMap, projectName)

	return &types.ResourceManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindCluster,
		Metadata:   metadata,
		Spec:       spec,
	}, nil
}

// convertDatabaseUserManifest converts a database user manifest from discovered format
func (c *DiscoveredProjectConverter) convertDatabaseUserManifest(userRaw interface{}, projectName string) (*types.ResourceManifest, error) {
	userMap, ok := userRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid database user manifest format")
	}

	metadata := c.extractMetadata(userMap, "database-user")
	spec := c.extractSpec(userMap, projectName)

	// DO NOT mask password for existing users - only remove password field entirely
	// This ensures existing users are not modified unless explicitly intended
	if specMap, ok := spec.(map[string]interface{}); ok {
		if _, hasPassword := specMap["password"]; hasPassword {
			// Remove password field completely to avoid modifying existing users
			delete(specMap, "password")
			metadata.Annotations["password-handling"] = "existing-user-password-preserved"
			metadata.Annotations["password-note"] = "Password field removed to preserve existing user credentials"
		}
	}

	return &types.ResourceManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindDatabaseUser,
		Metadata:   metadata,
		Spec:       spec,
	}, nil
}

// convertNetworkAccessManifest converts a network access manifest from discovered format
func (c *DiscoveredProjectConverter) convertNetworkAccessManifest(networkRaw interface{}, projectName string) (*types.ResourceManifest, error) {
	networkMap, ok := networkRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid network access manifest format")
	}

	metadata := c.extractMetadata(networkMap, "network-access")
	spec := c.extractSpec(networkMap, projectName)

	// Fix network access spec to avoid having both ipAddress and cidr
	if specMap, ok := spec.(map[string]interface{}); ok {
		// If we have both ipAddress and cidr, prefer ipAddress for single IPs
		if _, hasIP := specMap["ipAddress"].(string); hasIP {
			if cidr, hasCIDR := specMap["cidr"].(string); hasCIDR {
				// If CIDR is a /32 (single IP), remove cidr and keep ipAddress
				if strings.HasSuffix(cidr, "/32") {
					delete(specMap, "cidr")
				} else {
					// If CIDR is not /32, remove ipAddress and keep cidr
					delete(specMap, "ipAddress")
				}
			}
		}
	}

	return &types.ResourceManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindNetworkAccess,
		Metadata:   metadata,
		Spec:       spec,
	}, nil
}

// extractMetadata extracts metadata from a discovered resource
func (c *DiscoveredProjectConverter) extractMetadata(resourceMap map[string]interface{}, resourceType string) types.ResourceMetadata {
	metadata := types.ResourceMetadata{
		Labels:      map[string]string{},
		Annotations: map[string]string{},
	}

	// Extract existing metadata if present
	if metadataRaw, exists := resourceMap["metadata"]; exists {
		if metadataMap, ok := metadataRaw.(map[string]interface{}); ok {
			if name, ok := metadataMap["name"].(string); ok {
				// Don't sanitize network access names to preserve CIDR/IP format for matching
				if resourceType == "network-access" {
					metadata.Name = name
				} else {
					metadata.Name = c.sanitizeName(name)
				}
			}
			if labelsRaw, exists := metadataMap["labels"]; exists {
				if labelsMap, ok := labelsRaw.(map[string]interface{}); ok {
					for k, v := range labelsMap {
						if strValue, ok := v.(string); ok {
							// Use sanitized label keys for validation compliance
							if sanitizedKey := c.sanitizeLabelKey(k); sanitizedKey != "" {
								metadata.Labels[sanitizedKey] = strValue
							}
						}
					}
				}
			}
			if annotationsRaw, exists := metadataMap["annotations"]; exists {
				if annotationsMap, ok := annotationsRaw.(map[string]interface{}); ok {
					for k, v := range annotationsMap {
						if strValue, ok := v.(string); ok {
							metadata.Annotations[k] = strValue
						}
					}
				}
			}
		}
	}

	// Generate name if not present
	if metadata.Name == "" {
		if specMap, ok := resourceMap["spec"].(map[string]interface{}); ok {
			if name, ok := specMap["name"].(string); ok {
				metadata.Name = c.sanitizeName(name)
			} else if username, ok := specMap["username"].(string); ok {
				metadata.Name = c.sanitizeName(username)
			} else if cidr, ok := specMap["cidr"].(string); ok {
				// For network access, preserve original name format for matching
				metadata.Name = cidr
			} else if ipAddress, ok := specMap["ipAddress"].(string); ok {
				// For single IP, use the format that Atlas discovery uses (IP/32)
				metadata.Name = ipAddress + "/32"
			}
		}
		if metadata.Name == "" {
			metadata.Name = fmt.Sprintf("converted-%s-%d", resourceType, time.Now().Unix())
		}
	} else {
		// If we already have a name, don't sanitize it for network access resources
		// This preserves the original CIDR/IP format for proper matching
		if resourceType != "network-access" {
			metadata.Name = c.sanitizeName(metadata.Name)
		}
	}

	// Don't add conversion metadata for resources - this causes them to be seen as different
	// from the discovered originals when planning. Conversion metadata is only added
	// to the top-level ApplyDocument.

	return metadata
}

// extractSpec extracts the spec from a discovered resource and ensures projectName is set
func (c *DiscoveredProjectConverter) extractSpec(resourceMap map[string]interface{}, projectName string) interface{} {
	var spec map[string]interface{}

	if specRaw, exists := resourceMap["spec"]; exists {
		if specMap, ok := specRaw.(map[string]interface{}); ok {
			spec = specMap
		} else {
			spec = make(map[string]interface{})
		}
	} else {
		// If no spec field, create one from the resource (some discovered resources might not have spec wrapper)
		spec = make(map[string]interface{})
		for k, v := range resourceMap {
			if k != "apiVersion" && k != "kind" && k != "metadata" && k != "status" {
				spec[k] = v
			}
		}
	}

	// Only set projectName if it's completely missing (nil)
	// Don't modify existing projectName values, even if empty, to maintain idempotency
	if spec["projectName"] == nil {
		spec["projectName"] = projectName
	}

	return spec
}

// sanitizeName converts a name to be valid for matlas (lowercase, no invalid chars)
func (c *DiscoveredProjectConverter) sanitizeName(name string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(name)

	// Replace invalid characters with hyphens
	invalidChars := []string{".", "/", " ", "!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "+", "=", "[", "]", "{", "}", "|", "\\", ":", ";", "\"", "'", "<", ">", ",", "?", "~", "`"}
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "-")
	}

	// Remove consecutive hyphens
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Ensure name is not empty
	if sanitized == "" {
		sanitized = "unnamed-resource"
	}

	return sanitized
}

// sanitizeLabelKey converts Atlas label keys to valid matlas label format
func (c *DiscoveredProjectConverter) sanitizeLabelKey(key string) string {
	// Skip Atlas-specific label keys that contain dots
	if strings.Contains(key, ".") {
		// Convert atlas.mongodb.com/project-id to atlas-project-id
		sanitized := strings.ReplaceAll(key, ".", "-")
		sanitized = strings.ReplaceAll(sanitized, "/", "-")

		// Ensure it matches the pattern: ^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$
		if len(sanitized) > 63 {
			return "" // Skip if too long
		}

		// Make sure it starts and ends with alphanumeric
		if len(sanitized) > 0 {
			if !((sanitized[0] >= 'a' && sanitized[0] <= 'z') ||
				(sanitized[0] >= 'A' && sanitized[0] <= 'Z') ||
				(sanitized[0] >= '0' && sanitized[0] <= '9')) {
				return ""
			}
			lastChar := sanitized[len(sanitized)-1]
			if !((lastChar >= 'a' && lastChar <= 'z') ||
				(lastChar >= 'A' && lastChar <= 'Z') ||
				(lastChar >= '0' && lastChar <= '9')) {
				return ""
			}
		}

		return sanitized
	}

	return key
}
