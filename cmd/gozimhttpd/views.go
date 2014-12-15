package main

import (
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/akhenakh/gozim"
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
		log.Printf("302 from %s to %s\n", r.URL.Path, "zim/"+string(cr.Data))
		http.Redirect(w, r, "/zim/"+string(cr.Data), http.StatusFound)
	} else if cr.ResponseType == NoResponse {
		log.Printf("404 %s\n", r.URL.Path)
		http.NotFound(w, r)
	} else if cr.ResponseType == DataResponse {
		log.Printf("200 %s\n", r.URL.Path)
		w.Header().Set("Content-Type", cr.MimeType)
		// 15 days
		w.Header().Set("Cache-control", "public, max-age=1350000")
		w.Write(cr.Data)
	}
}

// the handler receiving http request
func zimHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path[5:]
	// lookup in the cache for a cached response
	if cr, iscached := cacheLookup(url); iscached {
		handleCachedResponse(cr, w, r)
		return

	} else {
		var a *zim.Article
		a = Z.GetPageNoIndex(url)

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

func homeHandler(w http.ResponseWriter, r *http.Request) {
	var index bool
	if *indexPath != "" {
		index = true
	}
	var mainURL string

	mainPage := Z.GetMainPage()
	var hasMainPage bool

	if mainPage != nil {
		hasMainPage = true
		mainURL = "/zim/" + mainPage.FullURL()
	}

	d := map[string]interface{}{
		"Path":        path.Base(*zimPath),
		"Count":       strconv.Itoa(int(Z.ArticleCount)),
		"IsIndexed":   index,
		"HasMainPage": hasMainPage,
		"MainURL":     mainURL,
	}
	tplHome.Execute(w, d)
}

func browseHandler(w http.ResponseWriter, r *http.Request) {
	Articles := make([]*zim.Article, 20)
	var pos int
	for i := 100; i < 120; i++ {
		offset := Z.GetOffsetAtURLIdx(uint32(i))
		Articles[pos] = Z.GetArticleAt(offset)
		pos++
	}
	d := map[string]interface{}{
		"Page":     0,
		"Articles": Articles,
	}
	tplBrowse.Execute(w, d)
}
