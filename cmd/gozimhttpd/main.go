package main

import (
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"

	"github.com/GeertJohan/go.rice"
	"github.com/akhenakh/gozim"
	"github.com/blevesearch/bleve"
	"github.com/golang/groupcache/lru"
)

//
type ResponseType int8

const (
	RedirectResponse ResponseType = iota
	DataResponse
	NoResponse
)

type CachedResponse struct {
	ResponseType ResponseType
	Data         []byte
	MimeType     string
}

var (
	zimPath    = flag.String("path", "", "path for the zim file")
	indexPath  = flag.String("indexPath", "", "path for the index file")
	mmap       = flag.Bool("mmap", false, "use mmap")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	Z     *zim.ZimReader
	Cache *lru.Cache
	idx   bool
	index bleve.Index

	tplHome   *template.Template
	tplBrowse *template.Template
)

func init() {
	tplBox := rice.MustFindBox("templates")

	homeString, err := tplBox.String("index.html")
	if err != nil {
		log.Fatal(err)
	}
	tplHome = template.Must(template.New("Home").Parse(homeString))

	browseString, err := tplBox.String("browse.html")
	if err != nil {
		log.Fatal(err)
	}
	tplBrowse = template.Must(template.New("Browse").Parse(browseString))
}

func main() {
	flag.Parse()
	if *zimPath == "" {
		panic(errors.New("provide a zim file path"))
	}

	if *mmap {
		log.Println("Using mmap")
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for _ = range c {
				pprof.StopCPUProfile()
				os.Exit(1)
			}
		}()
	}

	// Do we have an index ?
	if _, err := os.Stat(*indexPath); err == nil {
		log.Println("Found indexes")
		idx = true

		// open the db
		index, err = bleve.Open(*indexPath)
		if err != nil {
			panic(err)
		}
	}

	// assets
	box := rice.MustFindBox("static")
	fileServer := http.StripPrefix("/static/", http.FileServer(box.HTTPBox()))
	http.Handle("/static/", fileServer)

	// crompress wiki pages
	http.HandleFunc("/zim/", makeGzipHandler(zimHandler))
	z, err := zim.NewReader(*zimPath, *mmap)
	Z = z
	if err != nil {
		panic(err)
	}

	// tpl
	http.HandleFunc("/browse/", makeGzipHandler(browseHandler))
	http.HandleFunc("/", makeGzipHandler(homeHandler))

	// the need for a cache is absolute
	// a lots of urls will be called repeatedly, css, js ...
	// this is less important when using indexes
	Cache = lru.New(100)

	http.ListenAndServe(":8080", nil)

}
