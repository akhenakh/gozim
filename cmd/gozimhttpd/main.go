package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"

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
	path       = flag.String("path", "", "path for the zim file")
	indexPath  = flag.String("indexPath", "", "path for the index file")
	mmap       = flag.Bool("mmap", false, "use mmap")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	Z     *zim.ZimReader
	Cache *lru.Cache
	idx   bool
	index bleve.Index
)

func cacheLookup(url string) (*CachedResponse, bool) {
	if v, ok := Cache.Get(url); ok {
		c := v.(CachedResponse)
		return &c, ok
	}
	return nil, false
}

// dealing with cached response, responding directly
func handleCachedResponse(cr *CachedResponse, w http.ResponseWriter, r *http.Request) {
	if cr.ResponseType == RedirectResponse {
		fmt.Printf("302 from %s to %s\n", r.URL.Path, string(cr.Data))
		http.Redirect(w, r, "/"+string(cr.Data), http.StatusFound)
	} else if cr.ResponseType == NoResponse {
		fmt.Printf("404 %s\n", r.URL.Path)
		http.NotFound(w, r)
	} else if cr.ResponseType == DataResponse {
		fmt.Printf("200 %s\n", r.URL.Path)
		w.Header().Set("Content-Type", cr.MimeType)
		// 15 days
		w.Header().Set("Cache-control", "public, max-age=1350000")
		w.Write(cr.Data)
	}
}

// the handler receiving http request
func handler(w http.ResponseWriter, r *http.Request) {

	url := r.URL.Path[1:]
	// lookup in the cache for a cached response
	if cr, iscached := cacheLookup(url); iscached {
		handleCachedResponse(cr, w, r)
		return

	} else {
		var a *zim.Article
		a = Z.GetPageNoIndex(url)

		if a == nil && url == "index.html" || url == "" {
			a = Z.GetMainPage()
		}

		if a == nil {
			Cache.Add(url, CachedResponse{ResponseType: NoResponse})
		} else if a.EntryType == zim.RedirectEntry {
			Cache.Add(url, CachedResponse{
				ResponseType: RedirectResponse,
				Data:         []byte(a.RedirectTo.FullURL())})
		} else {
			Cache.Add(url, CachedResponse{
				ResponseType: DataResponse,
				Data:         a.Data(),
				MimeType:     a.MimeType(),
			})
		}

		// look again in the cache for the same entry
		if cr, iscached := cacheLookup(url); iscached {
			handleCachedResponse(cr, w, r)
		}
	}
}

func main() {
	flag.Parse()
	if *path == "" {
		panic(errors.New("provide a zim file path"))
	}

	if *mmap {
		fmt.Println("Using mmap")
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
		fmt.Println("Found indexes")
		idx = true

		// open the db
		index, err = bleve.Open(*indexPath)
		if err != nil {
			panic(err)
		}
	}

	http.HandleFunc("/", makeGzipHandler(handler))
	z, err := zim.NewReader(*path, *mmap)
	Z = z
	if err != nil {
		panic(err)
	}

	// the need for a cache is absolute
	// a lots of urls will be called repeatedly, css, js ...
	// this is less important when using indexes
	Cache = lru.New(60)

	http.ListenAndServe(":8080", nil)

}
