package zim

import (
	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/language/en"
	"github.com/blevesearch/bleve/analysis/token_filters/lower_case_filter"
	"github.com/blevesearch/bleve/analysis/tokenizers/unicode"
	"github.com/blevesearch/bleve/registry"
)

func AnalyzerConstructorEn(config map[string]interface{}, cache *registry.Cache) (*analysis.Analyzer, error) {
	tokenizer, err := cache.TokenizerNamed(unicode.Name)
	if err != nil {
		return nil, err
	}
	possEnFilter, err := cache.TokenFilterNamed(en.PossessiveName)
	if err != nil {
		return nil, err
	}
	toLowerFilter, err := cache.TokenFilterNamed(lower_case_filter.Name)
	if err != nil {
		return nil, err
	}
	stopEnFilter, err := cache.TokenFilterNamed(en.StopName)
	if err != nil {
		return nil, err
	}

	rv := analysis.Analyzer{
		Tokenizer: tokenizer,
		TokenFilters: []analysis.TokenFilter{
			possEnFilter,
			toLowerFilter,
			stopEnFilter,
		},
	}
	return &rv, nil
}

func init() {
	registry.RegisterAnalyzer("ennostemm", AnalyzerConstructorEn)
}
