package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/akhenakh/gozim"
	"github.com/blevesearch/bleve"
)

const (
	ArticlesPerPage = 16
)

func cacheLookup(url string) (*CachedResponse, bool) {
	if v, ok := cache.Get(url); ok {
		c := v.(CachedResponse)
		return &c, ok
	}
	return nil, false
}

// dealing with cached response, responding directly
func handleCachedResponse(cr *CachedResponse, w http.ResponseWriter, r *http.Request) {
	if cr.ResponseType == RedirectResponse {
		log.Printf("302 from %s to %s\n", r.URL.Path, "zim/"+string(cr.Data))
		http.Redirect(w, r, "/zim/"+string(cr.Data), http.StatusMovedPermanently)
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
		a, _ = Z.GetPageNoIndex(url)

		if a == nil {
			cache.Add(url, CachedResponse{ResponseType: NoResponse})
		} else if a.EntryType == zim.RedirectEntry {
			ridx, err := a.RedirectIndex()
			if err != nil {
				cache.Add(url, CachedResponse{ResponseType: NoResponse})
			} else {
				ra, err := Z.ArticleAtURLIdx(ridx)
				if err != nil {
					cache.Add(url, CachedResponse{ResponseType: NoResponse})
				} else {
					cache.Add(url, CachedResponse{
						ResponseType: RedirectResponse,
						Data:         []byte(ra.FullURL())})
				}
			}
		} else {
			data, err := a.Data()
			if err != nil {
				cache.Add(url, CachedResponse{ResponseType: NoResponse})
			} else {
				cache.Add(url, CachedResponse{
					ResponseType: DataResponse,
					Data:         data,
					MimeType:     a.MimeType(),
				})
			}
		}

		// look again in the cache for the same entry
		if cr, iscached := cacheLookup(url); iscached {
			handleCachedResponse(cr, w, r)
		}
	}
}

// homeHandler is displaying the / page but redirect every other requests to /zim/xxx
// some zim files have / hardcoded in their pages
func homeHandler(w http.ResponseWriter, r *http.Request) {
	var mainURL string

	if r.URL.Path != "/" {
		http.Redirect(w, r, "/zim"+r.URL.Path, http.StatusMovedPermanently)
		return
	}

	mainPage, err := Z.MainPage()
	var hasMainPage bool

	if err != nil && mainPage != nil {
		hasMainPage = true
		mainURL = "/zim/" + mainPage.FullURL()
	}

	d := map[string]interface{}{
		"Path":        path.Base(*zimPath),
		"Count":       strconv.Itoa(int(Z.ArticleCount)),
		"IsIndexed":   idx,
		"HasMainPage": hasMainPage,
		"MainURL":     mainURL,
	}
	templates["index"].Execute(w, d)
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {

	d := map[string]interface{}{}
	templates["about"].Execute(w, d)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	d := map[string]interface{}{
		"Path": path.Base(*zimPath),
	}

	if !idx {
		templates["searchNoIdx"].Execute(w, d)
		return
	}

	if r.Method == "GET" {
		templates["search"].Execute(w, d)
		return
	}

	q := r.FormValue("search_data")
	if q == "" {
		templates["search"].Execute(w, d)
		return
	}

	query := bleve.NewQueryStringQuery(q)
	search := bleve.NewSearchRequest(query)
	search.Fields = []string{"Title"}

	sr, err := index.Search(search)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if sr.Total > 0 {
		d["Info"] = fmt.Sprintf("%d matches for query [%s], took %s", sr.Total, q, sr.Took)

		// Constructs a list of Hits
		var l []map[string]string

		for _, h := range sr.Hits {
			idx, err := strconv.Atoi(h.ID)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			a, err := Z.ArticleAtURLIdx(uint32(idx))
			if err != nil {
				continue
			}
			l = append(l, map[string]string{
				"Score": strconv.FormatFloat(h.Score, 'f', 1, 64),
				"Title": a.Title,
				"URL":   "/zim/" + a.FullURL()})

		}
		d["Hits"] = l

	} else {
		d["Info"] = fmt.Sprintf("No match for [%s], took %s", q, sr.Took)
		d["Hits"] = 0
	}

	templates["searchResult"].Execute(w, d)
}

// browseHandler is browsing the zim page per page
func browseHandler(w http.ResponseWriter, r *http.Request) {
	var page, previousPage, nextPage int

	p := r.URL.Query().Get("page")
	if p != "" {
		page, _ = strconv.Atoi(p)
	}

	if page*ArticlesPerPage-1 >= int(Z.ArticleCount) {
		http.NotFound(w, r)
		return
	}

	Articles := make([]*zim.Article, ArticlesPerPage)
	var pos int
	for i := page * ArticlesPerPage; i < page*ArticlesPerPage+ArticlesPerPage; i++ {
		a, err := Z.ArticleAtURLIdx(uint32(i))
		if err != nil {
			continue
		}
		if a.Title == "" {
			a.Title = a.FullURL()
		}
		Articles[pos] = a
		pos++
	}

	if page == 0 {
		previousPage = 0
	} else {
		previousPage = page - 1
	}

	if page*ArticlesPerPage-1 >= int(Z.ArticleCount) {
		nextPage = page
	} else {
		nextPage = page + 1
	}

	d := map[string]interface{}{
		"Page":         page,
		"PreviousPage": previousPage,
		"NextPage":     nextPage,
		"Articles":     Articles,
	}
	templates["browse"].Execute(w, d)
}

func robotHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
}
