package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/akhenakh/gozim"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/registry"
)

type ArticleIndex struct {
	Title string
}

var (
	path       = flag.String("path", "", "path for the zim file")
	indexPath  = flag.String("index", "", "path for the index file")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	z          *zim.ZimReader
	lang       = flag.String("lang", "en", "language for indexation")
)

func inList(s []string, value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}

// Type return the Article type (used for bleve indexer)
func (a *ArticleIndex) Type() string {
	return "Article"
}

func main() {

	flag.Parse()

	if *path == "" {
		panic("provide a zim file path")
	}

	z, err := zim.NewReader(*path, false)
	if err != nil {
		panic(err)
	}

	if *indexPath == "" {
		panic("Please provide a path for the index")
	}

	switch *lang {
	case "en":
		//TODO: create a simple language support for stop word
	default:
		panic("unsupported language")
	}

	mapping := bleve.NewIndexMapping()
	mapping.DefaultType = "Article"

	articleMapping := bleve.NewDocumentMapping()
	mapping.AddDocumentMapping("Article", articleMapping)

	titleMapping := bleve.NewTextFieldMapping()
	titleMapping.Store = false
	titleMapping.Index = true
	titleMapping.Analyzer = "standard"
	articleMapping.AddFieldMappingsAt("Title", titleMapping)

	fmt.Println(registry.AnalyzerTypesAndInstances())

	index, err := bleve.New(*indexPath, mapping)
	if err != nil {
		panic(err)
	}

	i := 0

	batch := bleve.NewBatch()
	batchCount := 0
	idoc := ArticleIndex{}

	z.ListTitlesPtrIterator(func(idx uint32) {

		if i == 10000 {
			fmt.Print("*")
			i = 0
		}

		offset := z.OffsetAtURLIdx(idx)
		a := z.ArticleAt(offset)
		if a.EntryType == zim.RedirectEntry || a.EntryType == zim.LinkTargetEntry || a.EntryType == zim.DeletedEntry {
			return
		}

		if a.Namespace == 'A' {
			idoc.Title = a.Title
			// index the idoc with the idx as key
			batch.Index(fmt.Sprint(idx), idoc)
		}

		batchCount++
		i++

		// send a batch to bleve
		if batchCount >= 10000 {
			err = index.Batch(batch)
			if err != nil {
				log.Fatal(err.Error())
			}
			batch = bleve.NewBatch()
			batchCount = 0
		}
	})

	// batch the rest
	if batchCount > 0 {
		err = index.Batch(batch)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	index.Close()

}
