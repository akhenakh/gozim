package main

import (
	"flag"
	"fmt"

	"github.com/akhenakh/gozim"
	"github.com/blevesearch/bleve"
)

var (
	path       = flag.String("path", "", "path for the zim file")
	indexPath  = flag.String("indexPath", "", "path for the index file")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	z          *zim.ZimReader
)

func inList(s []string, value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
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

	mapping := bleve.NewIndexMapping()
	index, err := bleve.New(*indexPath, mapping)
	if err != nil {
		panic(err)
	}

	i := 0

	type IndexDoc struct {
		Title  string
		Offset uint64
	}

	for idx := range z.ListTitlesPtr() {
		offset := z.GetOffsetAtURLIdx(idx)
		a := z.GetArticleAt(offset)
		fmt.Println(a.Title)
		idoc := IndexDoc{Title: a.Title, Offset: offset}
		index.Index(idoc.Title, idoc)

		i++
		if i == 1000 {
			fmt.Print("*")
			i = 0
		}
	}
}
