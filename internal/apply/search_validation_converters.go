package apply

import "github.com/teabranch/matlas-cli/internal/types"

// convertToSearchMetricsSpec converts a map to SearchMetricsSpec
func convertToSearchMetricsSpec(specMap map[string]interface{}) types.SearchMetricsSpec {
	spec := types.SearchMetricsSpec{}

	if val, ok := specMap["projectName"].(string); ok {
		spec.ProjectName = val
	}
	if val, ok := specMap["clusterName"].(string); ok {
		spec.ClusterName = val
	}
	if val, ok := specMap["indexName"].(string); ok {
		spec.IndexName = &val
	}
	if val, ok := specMap["timeRange"].(string); ok {
		spec.TimeRange = val
	}
	if val, ok := specMap["metrics"].([]interface{}); ok {
		spec.Metrics = make([]string, len(val))
		for i, metric := range val {
			if metricStr, ok := metric.(string); ok {
				spec.Metrics[i] = metricStr
			}
		}
	}
	if val, ok := specMap["dependsOn"].([]interface{}); ok {
		spec.DependsOn = make([]string, len(val))
		for i, dep := range val {
			if depStr, ok := dep.(string); ok {
				spec.DependsOn[i] = depStr
			}
		}
	}

	return spec
}

// convertToSearchOptimizationSpec converts a map to SearchOptimizationSpec
func convertToSearchOptimizationSpec(specMap map[string]interface{}) types.SearchOptimizationSpec {
	spec := types.SearchOptimizationSpec{}

	if val, ok := specMap["projectName"].(string); ok {
		spec.ProjectName = val
	}
	if val, ok := specMap["clusterName"].(string); ok {
		spec.ClusterName = val
	}
	if val, ok := specMap["indexName"].(string); ok {
		spec.IndexName = &val
	}
	if val, ok := specMap["analyzeAll"].(bool); ok {
		spec.AnalyzeAll = val
	}
	if val, ok := specMap["categories"].([]interface{}); ok {
		spec.Categories = make([]string, len(val))
		for i, category := range val {
			if categoryStr, ok := category.(string); ok {
				spec.Categories[i] = categoryStr
			}
		}
	}
	if val, ok := specMap["dependsOn"].([]interface{}); ok {
		spec.DependsOn = make([]string, len(val))
		for i, dep := range val {
			if depStr, ok := dep.(string); ok {
				spec.DependsOn[i] = depStr
			}
		}
	}

	return spec
}

// convertToSearchQueryValidationSpec converts a map to SearchQueryValidationSpec
func convertToSearchQueryValidationSpec(specMap map[string]interface{}) types.SearchQueryValidationSpec {
	spec := types.SearchQueryValidationSpec{}

	if val, ok := specMap["projectName"].(string); ok {
		spec.ProjectName = val
	}
	if val, ok := specMap["clusterName"].(string); ok {
		spec.ClusterName = val
	}
	if val, ok := specMap["indexName"].(string); ok {
		spec.IndexName = val
	}
	if val, ok := specMap["query"].(map[string]interface{}); ok {
		spec.Query = val
	}
	if val, ok := specMap["testMode"].(bool); ok {
		spec.TestMode = val
	}
	if val, ok := specMap["validate"].([]interface{}); ok {
		spec.Validate = make([]string, len(val))
		for i, validate := range val {
			if validateStr, ok := validate.(string); ok {
				spec.Validate[i] = validateStr
			}
		}
	}
	if val, ok := specMap["dependsOn"].([]interface{}); ok {
		spec.DependsOn = make([]string, len(val))
		for i, dep := range val {
			if depStr, ok := dep.(string); ok {
				spec.DependsOn[i] = depStr
			}
		}
	}

	return spec
}
