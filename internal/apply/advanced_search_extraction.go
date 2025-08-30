package apply

import (
	"github.com/teabranch/matlas-cli/internal/types"
)

// extractAdvancedSearchFeatures extracts advanced search features from the Atlas search index definition
func (d *AtlasStateDiscovery) extractAdvancedSearchFeatures(definition interface{}, spec *types.SearchIndexSpec) {
	defMap, ok := definition.(map[string]interface{})
	if !ok {
		return
	}

	// Extract analyzers
	if analyzers := d.extractAnalyzers(defMap); len(analyzers) > 0 {
		spec.Analyzers = analyzers
	}

	// Extract facets from mappings
	if facets := d.extractFacets(defMap); len(facets) > 0 {
		spec.Facets = facets
	}

	// Extract autocomplete configurations
	if autocomplete := d.extractAutocomplete(defMap); len(autocomplete) > 0 {
		spec.Autocomplete = autocomplete
	}

	// Extract highlighting configurations
	if highlighting := d.extractHighlighting(defMap); len(highlighting) > 0 {
		spec.Highlighting = highlighting
	}

	// Extract synonyms
	if synonyms := d.extractSynonyms(defMap); len(synonyms) > 0 {
		spec.Synonyms = synonyms
	}

	// Extract fuzzy search configurations
	if fuzzy := d.extractFuzzySearch(defMap); len(fuzzy) > 0 {
		spec.FuzzySearch = fuzzy
	}
}

// extractAnalyzers extracts analyzer configurations from the search index definition
func (d *AtlasStateDiscovery) extractAnalyzers(defMap map[string]interface{}) []types.AnalyzerConfig {
	var analyzers []types.AnalyzerConfig

	// Check for custom analyzers in the definition
	if analyzersRaw, ok := defMap["analyzers"]; ok {
		if analyzersList, ok := analyzersRaw.([]interface{}); ok {
			for _, analyzerRaw := range analyzersList {
				if analyzerMap, ok := analyzerRaw.(map[string]interface{}); ok {
					analyzer := types.AnalyzerConfig{}
					if name, ok := analyzerMap["name"].(string); ok {
						analyzer.Name = name
					}
					if analyzerType, ok := analyzerMap["type"].(string); ok {
						analyzer.Type = analyzerType
					}
					if charFilters, ok := analyzerMap["charFilters"].([]interface{}); ok {
						analyzer.CharFilters = charFilters
					}
					if tokenizer, ok := analyzerMap["tokenizer"].(map[string]interface{}); ok {
						analyzer.Tokenizer = tokenizer
					}
					if tokenFilters, ok := analyzerMap["tokenFilters"].([]interface{}); ok {
						analyzer.TokenFilters = tokenFilters
					}
					analyzers = append(analyzers, analyzer)
				}
			}
		}
	}

	return analyzers
}

// extractFacets extracts facet configurations from the search index definition
func (d *AtlasStateDiscovery) extractFacets(defMap map[string]interface{}) []types.FacetConfig {
	var facets []types.FacetConfig

	// Check for facets in mappings
	if mappings, ok := defMap["mappings"].(map[string]interface{}); ok {
		if fields, ok := mappings["fields"].(map[string]interface{}); ok {
			for fieldName, fieldValue := range fields {
				if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
					if facetConfig := d.extractFieldFacet(fieldName, fieldMap); facetConfig != nil {
						facets = append(facets, *facetConfig)
					}
				}
			}
		}
	}

	return facets
}

// extractFieldFacet extracts facet configuration from a field definition
func (d *AtlasStateDiscovery) extractFieldFacet(fieldName string, fieldMap map[string]interface{}) *types.FacetConfig {
	// Check if this field has facet configuration
	if facetRaw, ok := fieldMap["facet"]; ok {
		if facetMap, ok := facetRaw.(map[string]interface{}); ok {
			facet := &types.FacetConfig{
				Field: fieldName,
			}

			if facetType, ok := facetMap["type"].(string); ok {
				facet.Type = facetType
			}
			if numBuckets, ok := facetMap["numBuckets"].(float64); ok {
				buckets := int(numBuckets)
				facet.NumBuckets = &buckets
			}
			if boundaries, ok := facetMap["boundaries"].([]interface{}); ok {
				facet.Boundaries = boundaries
			}
			if defaultVal, ok := facetMap["default"].(string); ok {
				facet.Default = &defaultVal
			}

			return facet
		}
	}
	return nil
}

// extractAutocomplete extracts autocomplete configurations from the search index definition
func (d *AtlasStateDiscovery) extractAutocomplete(defMap map[string]interface{}) []types.AutocompleteConfig {
	var autocomplete []types.AutocompleteConfig

	// Check for autocomplete in mappings
	if mappings, ok := defMap["mappings"].(map[string]interface{}); ok {
		if fields, ok := mappings["fields"].(map[string]interface{}); ok {
			for fieldName, fieldValue := range fields {
				if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
					if autocompleteConfig := d.extractFieldAutocomplete(fieldName, fieldMap); autocompleteConfig != nil {
						autocomplete = append(autocomplete, *autocompleteConfig)
					}
				}
			}
		}
	}

	return autocomplete
}

// extractFieldAutocomplete extracts autocomplete configuration from a field definition
func (d *AtlasStateDiscovery) extractFieldAutocomplete(fieldName string, fieldMap map[string]interface{}) *types.AutocompleteConfig {
	// Check if this field has autocomplete configuration
	if autocompleteRaw, ok := fieldMap["autocomplete"]; ok {
		if autocompleteMap, ok := autocompleteRaw.(map[string]interface{}); ok {
			autoComplete := &types.AutocompleteConfig{
				Field: fieldName,
			}

			if maxEdits, ok := autocompleteMap["maxEdits"].(float64); ok {
				autoComplete.MaxEdits = int(maxEdits)
			}
			if prefixLength, ok := autocompleteMap["prefixLength"].(float64); ok {
				autoComplete.PrefixLength = int(prefixLength)
			}
			if fuzzyMaxEdits, ok := autocompleteMap["fuzzyMaxEdits"].(float64); ok {
				autoComplete.FuzzyMaxEdits = int(fuzzyMaxEdits)
			}

			return autoComplete
		}
	}
	return nil
}

// extractHighlighting extracts highlighting configurations from the search index definition
func (d *AtlasStateDiscovery) extractHighlighting(defMap map[string]interface{}) []types.HighlightingConfig {
	var highlighting []types.HighlightingConfig

	// Check for highlighting in mappings
	if mappings, ok := defMap["mappings"].(map[string]interface{}); ok {
		if fields, ok := mappings["fields"].(map[string]interface{}); ok {
			for fieldName, fieldValue := range fields {
				if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
					if highlightConfig := d.extractFieldHighlighting(fieldName, fieldMap); highlightConfig != nil {
						highlighting = append(highlighting, *highlightConfig)
					}
				}
			}
		}
	}

	return highlighting
}

// extractFieldHighlighting extracts highlighting configuration from a field definition
func (d *AtlasStateDiscovery) extractFieldHighlighting(fieldName string, fieldMap map[string]interface{}) *types.HighlightingConfig {
	// Check if this field has highlighting configuration
	if highlightRaw, ok := fieldMap["highlight"]; ok {
		if highlightMap, ok := highlightRaw.(map[string]interface{}); ok {
			highlight := &types.HighlightingConfig{
				Field: fieldName,
			}

			if maxChars, ok := highlightMap["maxCharsToExamine"].(float64); ok {
				highlight.MaxCharsToExamine = int(maxChars)
			}
			if maxPassages, ok := highlightMap["maxNumPassages"].(float64); ok {
				highlight.MaxNumPassages = int(maxPassages)
			}

			return highlight
		}
	}
	return nil
}

// extractSynonyms extracts synonym configurations from the search index definition
func (d *AtlasStateDiscovery) extractSynonyms(defMap map[string]interface{}) []types.SynonymConfig {
	var synonyms []types.SynonymConfig

	// Check for synonyms in the definition
	if synonymsRaw, ok := defMap["synonyms"]; ok {
		if synonymsList, ok := synonymsRaw.([]interface{}); ok {
			for _, synonymRaw := range synonymsList {
				if synonymMap, ok := synonymRaw.(map[string]interface{}); ok {
					synonym := types.SynonymConfig{}
					if name, ok := synonymMap["name"].(string); ok {
						synonym.Name = name
					}
					if input, ok := synonymMap["input"].([]interface{}); ok {
						synonym.Input = make([]string, len(input))
						for i, inp := range input {
							if str, ok := inp.(string); ok {
								synonym.Input[i] = str
							}
						}
					}
					if output, ok := synonymMap["output"].(string); ok {
						synonym.Output = output
					}
					if explicit, ok := synonymMap["explicit"].(bool); ok {
						synonym.Explicit = explicit
					}
					synonyms = append(synonyms, synonym)
				}
			}
		}
	}

	return synonyms
}

// extractFuzzySearch extracts fuzzy search configurations from the search index definition
func (d *AtlasStateDiscovery) extractFuzzySearch(defMap map[string]interface{}) []types.FuzzyConfig {
	var fuzzy []types.FuzzyConfig

	// Check for fuzzy search in mappings
	if mappings, ok := defMap["mappings"].(map[string]interface{}); ok {
		if fields, ok := mappings["fields"].(map[string]interface{}); ok {
			for fieldName, fieldValue := range fields {
				if fieldMap, ok := fieldValue.(map[string]interface{}); ok {
					if fuzzyConfig := d.extractFieldFuzzy(fieldName, fieldMap); fuzzyConfig != nil {
						fuzzy = append(fuzzy, *fuzzyConfig)
					}
				}
			}
		}
	}

	return fuzzy
}

// extractFieldFuzzy extracts fuzzy search configuration from a field definition
func (d *AtlasStateDiscovery) extractFieldFuzzy(fieldName string, fieldMap map[string]interface{}) *types.FuzzyConfig {
	// Check if this field has fuzzy configuration
	if fuzzyRaw, ok := fieldMap["fuzzy"]; ok {
		if fuzzyMap, ok := fuzzyRaw.(map[string]interface{}); ok {
			fuzzyConfig := &types.FuzzyConfig{
				Field: fieldName,
			}

			if maxEdits, ok := fuzzyMap["maxEdits"].(float64); ok {
				fuzzyConfig.MaxEdits = int(maxEdits)
			}
			if prefixLength, ok := fuzzyMap["prefixLength"].(float64); ok {
				fuzzyConfig.PrefixLength = int(prefixLength)
			}
			if maxExpansions, ok := fuzzyMap["maxExpansions"].(float64); ok {
				fuzzyConfig.MaxExpansions = int(maxExpansions)
			}

			return fuzzyConfig
		}
	}
	return nil
}
