package zim

import "testing"

var Z *ZimReader

func setup(t *testing.T) {
	if Z == nil {
		z, err := NewReader("test.zim")

		if err != nil {
			t.Errorf("Can't read %v", err)
		}
		Z = z
	}

}

func TestOpen(t *testing.T) {
	setup(t)
	if Z.ArticleCount == 0 {
		t.Errorf("No article found")
	}
}

func TestMime(t *testing.T) {
	setup(t)

	if len(Z.MimeTypes()) == 0 {
		t.Errorf("No mime types found")
	}
}

func TestDisplayInfost(t *testing.T) {
	setup(t)
	info := Z.String()
	if len(info) < 0 {
		t.Errorf("Can't read infos")
	}
	t.Log(info)
}

func TestGetUrlAtIdx(t *testing.T) {
	setup(t)

	// addr 0 is a redirect
	p := Z.GetUrlOffsetAtIdx(4999)
	if Z.getArticleAt(p) == nil {
		t.Errorf("Can't find 1st url")
	}
}

func TestDisplayArticle(t *testing.T) {
	setup(t)

	// addr 0 is a redirect
	p := Z.GetUrlOffsetAtIdx(4999)

	var a *Article
	if a = Z.getArticleAt(p); a == nil {
		t.Errorf("Can't find 1st url")
	}

	t.Log(a)
}

func TestListArticles(t *testing.T) {
	setup(t)

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
	setup(t)

	var a *Article
	if a = Z.GetMainPage(); a == nil {
		t.Errorf("Can't find the mainpage article")
	}

	t.Log(a)
}

func TestData(t *testing.T) {
	setup(t)

	// addr 0 is a redirect
	p := Z.GetUrlOffsetAtIdx(2522170)
	a := Z.getArticleAt(p)
	data := string(a.Data(Z))
	if a.Mimetype != RedirectEntry {
		if len(data) == 0 {
			t.Error("can't read data")
		}
	}
	t.Log(a.String())
	t.Log(data)
}
