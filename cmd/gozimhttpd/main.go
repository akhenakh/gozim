package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"runtime"

	"github.com/akhenakh/gozim"
	"github.com/golang/groupcache/lru"
)

type ResponseType uint16

const (
	RedirectResponse ResponseType = 0xffff
	DataResponse                  = 0x0000
	NoResponse                    = 0x0404
)

type CachedResponse struct {
	ResponseType ResponseType
	Data         []byte
	MimeType     string
}

var (
	path  = flag.String("path", "", "path for the zim file")
	Z     *zim.ZimReader
	Cache *lru.Cache
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

	if cr, iscached := cacheLookup(url); iscached {
		handleCachedResponse(cr, w, r)
		return

	} else {
		var a *zim.Article

		if url == "index.html" {
			a = Z.GetMainPage()
		} else {
			a = Z.GetPageNoIndex(url)
		}

		if a == nil {
			Cache.Add(r.URL.Path[1:], CachedResponse{ResponseType: NoResponse})
		} else if a.Mimetype == zim.RedirectEntry {
			Cache.Add(r.URL.Path[1:], CachedResponse{
				ResponseType: RedirectResponse,
				Data:         []byte(a.RedirectTo.FullURL())})
		} else {
			Cache.Add(r.URL.Path[1:], CachedResponse{
				ResponseType: DataResponse,
				Data:         a.Data(Z),
				MimeType:     Z.MimeTypes()[a.Mimetype],
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
		panic(errors.New("Provide a zim file path"))

	}
	http.HandleFunc("/", handler)
	z, err := zim.NewReader(*path)
	Z = z
	if err != nil {
		panic(err)
	}

	fmt.Println(runtime.GOARCH)

	// the need of a cache is absolute
	// a lots of urls will be called repeatedly, css, js ...
	Cache = lru.New(30)

	http.ListenAndServe(":8080", nil)

}
