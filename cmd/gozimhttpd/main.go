package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"

	"github.com/akhenakh/gozim"
	"github.com/golang/groupcache/lru"
	"github.com/szferi/gomdb"
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
	mmap       = flag.Bool("mmap", false, "use mmap")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	Z     *zim.ZimReader
	Cache *lru.Cache
	idx   bool
	env   *mdb.Env
	dbi   mdb.DBI
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
		if idx {
			txn, _ := env.BeginTxn(nil, mdb.RDONLY)
			defer txn.Abort()
			b, _ := txn.Get(dbi, []byte(url))
			if len(b) != 0 {
				var v uint64
				buf := bytes.NewBuffer(b)
				err := binary.Read(buf, binary.LittleEndian, &v)
				if err == nil {
					a = Z.GetArticleAt(v)
				}
			}
		} else {
			a = Z.GetPageNoIndex(url)
		}

		if a == nil && url == "index.html" {
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
	pathidx := *path + "idx"
	if _, err := os.Stat(pathidx); err == nil {
		fmt.Println("Found indexes")
		idx = true

		// open the db
		env, err = mdb.NewEnv()
		if err != nil {
			panic(err)
		}

		err = env.Open(pathidx, 0, 0664)
		if err != nil {
			panic(err)
		}
		txn, _ := env.BeginTxn(nil, 0)
		dbi, _ = txn.DBIOpen(nil, 0)
	}

	http.HandleFunc("/", handler)
	z, err := zim.NewReader(*path, *mmap)
	Z = z
	if err != nil {
		panic(err)
	}

	// the need for a cache is absolute
	// a lots of urls will be called repeatedly, css, js ...
	// this is less important when using indexes
	Cache = lru.New(40)

	http.ListenAndServe(":8080", nil)

}
