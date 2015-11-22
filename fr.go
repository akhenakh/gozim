package zim

import (
	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/language/fr"
	"github.com/blevesearch/bleve/analysis/token_filters/lower_case_filter"
	"github.com/blevesearch/bleve/analysis/tokenizers/unicode"
	"github.com/blevesearch/bleve/registry"
)

func AnalyzerConstructorFr(config map[string]interface{}, cache *registry.Cache) (*analysis.Analyzer, error) {
	tokenizer, err := cache.TokenizerNamed(unicode.Name)
	if err != nil {
		return nil, err
	}
	elisionFilter, err := cache.TokenFilterNamed(fr.ElisionName)
	if err != nil {
		return nil, err
	}
	toLowerFilter, err := cache.TokenFilterNamed(lower_case_filter.Name)
	if err != nil {
		return nil, err
	}
	stopFrFilter, err := cache.TokenFilterNamed(fr.StopName)
	if err != nil {
		return nil, err
	}

	rv := analysis.Analyzer{
		Tokenizer: tokenizer,
		TokenFilters: []analysis.TokenFilter{
			toLowerFilter,
			elisionFilter,
			stopFrFilter,
		},
	}
	return &rv, nil
}

func init() {
	registry.RegisterAnalyzer("frnostemm", AnalyzerConstructorFr)
}
