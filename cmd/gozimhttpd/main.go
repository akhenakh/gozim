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
	"strconv"

	"github.com/GeertJohan/go.rice"
	"github.com/akhenakh/gozim"
	"github.com/blevesearch/bleve"
	"github.com/golang/groupcache/lru"
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
	Cache *lru.Cache
	idx   bool
	index bleve.Index

	templates map[string]*template.Template
)

func init() {
	templates = make(map[string]*template.Template)

	tplBox := rice.MustFindBox("templates")

	registerTemplate("index", tplBox)
	registerTemplate("browse", tplBox)
	registerTemplate("search", tplBox)
	registerTemplate("searchNoIdx", tplBox)
	registerTemplate("searchResult", tplBox)
	registerTemplate("about", tplBox)
}

// registerTemplate load template from rice box and add them to a map[string] call templates
func registerTemplate(name string, tplBox *rice.Box) {
	tplString, err := tplBox.String(name + ".html")
	if err != nil {
		log.Fatal(err)
	}
	templates[name] = template.Must(template.New(name).Parse(tplString))
}

func main() {
	flag.Parse()
	if *zimPath == "" {
		log.Fatal(errors.New("provide a zim file path"))
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
			log.Fatal(err)
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
		log.Fatal(err)
	}

	// tpl
	http.HandleFunc("/search/", makeGzipHandler(searchHandler))
	http.HandleFunc("/browse/", makeGzipHandler(browseHandler))
	http.HandleFunc("/about/", makeGzipHandler(aboutHandler))
	http.HandleFunc("/robots.txt", robotHandler)
	http.HandleFunc("/", makeGzipHandler(homeHandler))

	// the need for a cache is absolute
	// a lots of urls will be called repeatedly, css, js ...
	// this is less important when using indexes
	Cache = lru.New(40)

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
