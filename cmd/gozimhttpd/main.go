package main

import (
	"fmt"
	"net/http"

	"github.com/akhenakh/gozim"
)

var Z *zim.ZimReader

func handler(w http.ResponseWriter, r *http.Request) {
	var a *zim.Article

	if r.URL.Path == "/index.html" {
		a = Z.GetMainPage()
	} else {
		a = Z.GetPageNoIndex(r.URL.Path[1:])
	}

	if a == nil {
		fmt.Printf("404 %s\n", a.FullURL())

		http.NotFound(w, r)
		return
	}

	if a.Mimetype == zim.RedirectEntry {
		fmt.Printf("302 %s to %s\n", r.URL.Path, a.RedirectTo.URL)
		http.Redirect(w, r, a.RedirectTo.URL, http.StatusFound)
		return
	}
	fmt.Printf("200 %s\n", a.FullURL())
	w.Header().Set("Content-Type", Z.MimeTypes()[a.Mimetype])
	w.Write(a.Data(Z))
}

func main() {
	http.HandleFunc("/", handler)
	z, err := zim.NewReader("test.zim")
	Z = z
	if err != nil {
		panic(err)
	}

	http.ListenAndServe(":8080", nil)

}
