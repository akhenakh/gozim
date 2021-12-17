package main

import (
	"embed"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"

	zim "github.com/akhenakh/gozim"
	"github.com/blevesearch/bleve/v2"
	lru "github.com/hashicorp/golang-lru"

	_ "github.com/blevesearch/bleve/v2/analysis/lang/ar"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ckb"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/en"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/fa"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/fr"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/hi"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/it"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/pt"
)

type ResponseType int8

const (
	RedirectResponse ResponseType = iota
	DataResponse
	NoResponse
)

// CachedResponse cache the answer to an URL in the zim
type CachedResponse struct {
	ResponseType ResponseType
	Data         []byte
	MimeType     string
}

var (
	port       = flag.Int("port", -1, "port to listen to, read HOST env if not specified, default to 8080 otherwise")
	zimPath    = flag.String("path", "", "path for the zim file")
	indexPath  = flag.String("index", "", "path for the index file")
	mmap       = flag.Bool("mmap", false, "use mmap")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	Z *zim.ZimReader
	// Cache is filled with CachedResponse to avoid hitting the zim file for a zim URL
	cache *lru.ARCCache
	idx   bool
	index bleve.Index

	templates *template.Template

	//go:embed static
	staticFS embed.FS

	//go:embed templates/*
	templateFS embed.FS
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Parse()
	if *zimPath == "" {
		log.Fatal("provide a zim file path")
	}

	if *mmap {
		log.Println("Using mmap")
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				pprof.StopCPUProfile()
				os.Exit(1)
			}
		}()
	}

	// Do we have an index ?
	if indexPath != nil && *indexPath != "" {
		if _, err := os.Stat(*indexPath); err != nil {
			log.Fatal(err)
		}

		idx = true

		// open the db
		var err error
		index, err = bleve.Open(*indexPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	tpls, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	templates = tpls

	// static file handler
	fileServer := http.FileServer(http.FS(staticFS))
	http.Handle("/static/", fileServer)

	// compress wiki pages
	http.HandleFunc("/zim/", makeGzipHandler(zimHandler))
	z, err := zim.NewReader(*zimPath, *mmap)
	Z = z
	if err != nil {
		log.Fatal(err)
	}

	// tpl
	http.HandleFunc("/search/", makeGzipHandler(searchHandler))
	http.HandleFunc("/browse/", makeGzipHandler(browseHandler))
	http.HandleFunc("/about/", makeGzipHandler(aboutHandler))
	http.HandleFunc("/robots.txt", robotHandler)
	http.HandleFunc("/", makeGzipHandler(homeHandler))

	// the need for a cache is absolute
	// a lot of the same urls will be called repeatedly, css, js ...
	// avoid to look for those one
	cache, _ = lru.NewARC(40)

	// default listening to port 8080
	listenPath := ":8080"

	if len(os.Getenv("PORT")) > 0 {
		listenPath = ":" + os.Getenv("PORT")
	}

	if port != nil && *port > 0 {
		listenPath = ":" + strconv.Itoa(*port)
	}

	// Opening large indexes could takes minutes on raspberry
	log.Println("Listening on", listenPath)

	err = http.ListenAndServe(listenPath, nil)
	if err != nil {
		log.Fatal(err)
	}
}
