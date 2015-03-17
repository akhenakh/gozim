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
	indexPath  = flag.String("index", "", "path for the index directory")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	z          *zim.ZimReader
	lang       = flag.String("lang", "en", "language for indexation")
	batchSize  = flag.Int("batchsize", 10000, "size of bleve batches")
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
		log.Fatal("provide a zim file path")
	}

	z, err := zim.NewReader(*path, false)
	if err != nil {
		log.Fatal(err)
	}

	if *indexPath == "" {
		log.Fatal("Please provide a path for the index")
	}

	switch *lang {
	case "en":
		//TODO: create a simple language support for stop word
	default:
		log.Fatal("unsupported language")
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
		log.Fatal(err)
	}

	i := 0

	batch := index.NewBatch()
	batchCount := 0
	idoc := ArticleIndex{}

	divisor := float64(z.ArticleCount) / 100

	z.ListTitlesPtrIterator(func(idx uint32) {

		if i%*batchSize == 0 {
			fmt.Printf("%.2f%% done\n", float64(i)/divisor)
		}
		a, err := z.ArticleAtURLIdx(idx)
		if err != nil || a.EntryType == zim.DeletedEntry {
			i++
			return
		}

		if a.Namespace == 'A' {
			idoc.Title = a.Title
			// index the idoc with the idx as key
			batch.Index(fmt.Sprint(idx), idoc)
			batchCount++
		}

		i++

		// send a batch to bleve
		if batchCount >= *batchSize {
			err = index.Batch(batch)
			if err != nil {
				log.Fatal(err.Error())
			}
			batch = index.NewBatch()
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
