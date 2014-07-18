package zim

import "testing"

func TestOpen(t *testing.T) {
	z, err := NewReader("test.zim")
	defer z.Close()

	if err != nil {
		t.Errorf("Can't read %v", err)
	}

	if z.ArticleCount == 0 {
		t.Errorf("No article found")
	}
}

func TestMime(t *testing.T) {
	z, err := NewReader("test.zim")
	defer z.Close()

	if err != nil {
		t.Errorf("Can't read %v", err)
	}

	if len(z.MimeTypes()) == 0 {
		t.Errorf("No mime types found")
	}
}

func TestDisplayInfost(t *testing.T) {
	z, err := NewReader("test.zim")
	defer z.Close()
	if err != nil {
		t.Errorf("Can't read %v", err)
	}

	t.Log(z.String())
}

func TestGetUrlAtIdx(t *testing.T) {
	z, err := NewReader("test.zim")
	defer z.Close()
	if err != nil {
		t.Errorf("Can't read %v", err)
	}

	z.GetUrlAtIdx(0)
}

func TestListUrls(t *testing.T) {
	z, err := NewReader("test.zim")
	defer z.Close()
	if err != nil {
		t.Errorf("Can't read %v", err)
	}

	var i int

	for _ = range z.ListUrls() {
		i++
	}

	if i == 0 {
		t.Errorf("Can't find any urls")
	}
}
