package main

import (
	"fmt"
	"net/http"

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
	Z     *zim.ZimReader
	Cache *lru.Cache
)

func handler(w http.ResponseWriter, r *http.Request) {

cache:
	if v, ok := Cache.Get(r.URL.Path[1:]); ok {
		c := v.(CachedResponse)
		if c.ResponseType == RedirectResponse {
			fmt.Printf("302 from %s to %s\n", r.URL.Path, string(c.Data))
			http.Redirect(w, r, string(c.Data), http.StatusFound)
			return
		} else if c.ResponseType == NoResponse {
			fmt.Printf("404 %s\n", r.URL.Path)
			http.NotFound(w, r)
			return
		} else if c.ResponseType == DataResponse {
			fmt.Printf("200 %s\n", r.URL.Path)
			w.Header().Set("Content-Type", c.MimeType)
			w.Write(c.Data)
			return
		}
		return

	} else {
		var a *zim.Article

		if r.URL.Path[1:] == "index.html" {
			a = Z.GetMainPage()
		} else {
			a = Z.GetPageNoIndex(r.URL.Path[1:])
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

		goto cache
	}
}

func main() {
	http.HandleFunc("/", handler)
	z, err := zim.NewReader("test.zim")
	Z = z
	if err != nil {
		panic(err)
	}

	// the need of a cache is absolute
	// a lots of urls will be called repeatedly, css, js ...
	Cache = lru.New(30)

	http.ListenAndServe(":8080", nil)

}
