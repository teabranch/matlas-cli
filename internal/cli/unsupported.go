// Package cli provides command-line user experience helpers (errors, completion, recovery).
package cli

import "fmt"

// UnsupportedFeatureError returns a standardized error for unsupported features.
// Provide the feature name and, optionally, a brief detail hint.
func UnsupportedFeatureError(feature string, details ...string) error {
	base := fmt.Sprintf("%s is not yet supported in this build.", feature)
	if len(details) > 0 && details[0] != "" {
		return fmt.Errorf(base+" %s", details[0])
	}
	return fmt.Errorf("%s", base)
}

// UnsupportedSearchAPIError returns a consistent, helpful multi-line message for Atlas Search.
func UnsupportedSearchAPIError() error {
	return fmt.Errorf(`atlas search indexes are not yet supported by the Atlas Go SDK.

This feature will be available when the Atlas SDK includes the following APIs:
- AtlasSearchApi.ListAtlasSearchIndexes()
- AtlasSearchApi.GetAtlasSearchIndex()
- AtlasSearchApi.CreateAtlasSearchIndex()
- AtlasSearchApi.UpdateAtlasSearchIndex()
- AtlasSearchApi.DeleteAtlasSearchIndex()

For now, you can manage atlas search indexes through:
- MongoDB Atlas UI (https://cloud.mongodb.com)
- Official Atlas CLI (https://www.mongodb.com/docs/atlas/cli/)
- Direct Atlas API calls`)
}
