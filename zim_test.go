package zim

import (
	"log"
	"testing"
)

var Z *ZimReader

func init() {
	var err error
	Z, err = NewReader("test.zim", false)
	if err != nil {
		log.Panicf("Can't read %v", err)
	}
}

func TestOpen(t *testing.T) {
	if Z.ArticleCount == 0 {
		t.Errorf("No article found")
	}
}

func TestOpenMmap(t *testing.T) {
	z, err := NewReader("test.zim", true)

	if err != nil {
		t.Errorf("Can't read %v", err)
	}
	if z.ArticleCount == 0 {
		t.Errorf("No article found")
	}

	z.Close()
}

func TestMime(t *testing.T) {

	if len(Z.MimeTypes()) == 0 {
		t.Errorf("No mime types found")
	}
}

func TestDisplayInfost(t *testing.T) {
	info := Z.String()
	if len(info) < 0 {
		t.Errorf("Can't read infos")
	}
	t.Log(info)
}

func TestURLAtIdx(t *testing.T) {

	// addr 0 is a redirect
	p, _ := Z.OffsetAtURLIdx(5)
	a, _ := Z.ArticleAt(p)
	if a == nil {
		t.Errorf("Can't find 1st url")
	}
}

func TestDisplayArticle(t *testing.T) {

	// addr 0 is a redirect
	p, _ := Z.OffsetAtURLIdx(5)
	a, _ := Z.ArticleAt(p)
	if a == nil {
		t.Errorf("Can't find 1st url")
	}

	t.Log(a)
}

func TestPageNoIndex(t *testing.T) {

	a, _ := Z.GetPageNoIndex("A/Dracula:Capitol_1.html")
	if a == nil {
		t.Errorf("Can't find existing url")
	}
}

func TestListArticles(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var i uint32

	for a := range Z.ListArticles() {
		i++
		t.Log(a.String())
	}

	if i == 0 {
		t.Errorf("Can't find any urls")
	}

	if i != Z.ArticleCount-1 {
		t.Errorf("Can't find the exact ArticleCount urls %d vs %d", i, Z.ArticleCount)
	}
}

func TestMainPage(t *testing.T) {

	a, _ := Z.MainPage()
	if a == nil {
		t.Errorf("Can't find the mainpage article")
	}

	t.Log(a)
}

func TestData(t *testing.T) {

	// addr 0 is a redirect
	p, _ := Z.OffsetAtURLIdx(2)
	a, _ := Z.ArticleAt(p)
	b, _ := a.Data()
	data := string(b)
	if a.EntryType != RedirectEntry {
		if len(data) == 0 {
			t.Error("can't read data")
		}
	}
	t.Log(a.String())
	t.Log(data)
}

func BenchmarkArticleBytes(b *testing.B) {

	// addr 0 is a redirect
	p, _ := Z.OffsetAtURLIdx(5)
	a, _ := Z.ArticleAt(p)
	if a == nil {
		b.Errorf("Can't find 1st url")
	}
	data, err := a.Data()
	if err != nil {
		b.Error(err)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Data()
		bcache.Purge() // prevent memiozing value
	}

}
