package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

var (
	_ site = &lomb24{}
	_ page = &lomb24page{}
	_ page = &lomb24startpage{}
)

type lomb24startpage struct {
	body  []byte
	psite *lomb24
}

func (l *lomb24startpage) site() site {
	return l.psite
}

func (l *lomb24startpage) pages() ([]page, error) {
	var pages int
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(l.body))
	if err != nil {
		return nil, fmt.Errorf("cannot use gquery: %v", err)
	}
	doc.Find(l.psite.path).Each(func(i int, s *goquery.Selection) {
		pages++
	})
	ps := make([]page, pages)
	for i := 0; i < pages; i++ {
		ps[i] = &lomb24page{
			page:  i + 1,
			psite: l.psite,
		}
	}
	return ps, nil
}

func (l *lomb24startpage) contents() []byte {
	return l.body
}

func (l *lomb24startpage) url() string {
	return l.psite.startURL
}

type lomb24page struct {
	page  int
	body  []byte
	psite *lomb24
}

func (l *lomb24page) url() string {
	return fmt.Sprintf("lombard24.ee page %d", l.page)
}

func (l *lomb24page) site() site {
	return l.psite
}

func (l *lomb24page) contents() []byte {
	return l.body
}

func (l *lomb24page) pages() ([]page, error) {
	return nil, nil
}

type lomb24 struct {
	startURL, jsonURL string
	path              string
	root              *url.URL
}

func newLomb24() (*lomb24, error) {
	var err error
	l := &lomb24{
		startURL: "http://lombard24.ee/",
		jsonURL:  "http://lombard24.ee/addons/234jfsdm2n_lombard/ajax/change_page.php",
		path:     "#pg a",
	}
	l.root, err = url.Parse(l.startURL)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (l *lomb24) start(b *browser) (page, error) {
	body, err := b.get(l.startURL)
	if err != nil {
		return nil, err
	}
	sp := &lomb24startpage{body: body, psite: l}
	return sp, nil
}

func (l *lomb24) visit(p page, b *browser) (page, error) {
	ps, ok := p.(*lomb24startpage)
	if ok {
		body, err := b.get(ps.url())
		if err != nil {
			return nil, fmt.Errorf("cannot open startpage: %v", err)
		}
		ps.body = body
		return ps, nil
	}
	pg, ok := p.(*lomb24page)
	if !ok {
		return nil, errors.New("unexpected page")
	}
	data := url.Values{}
	data.Set("page", fmt.Sprintf("%d", pg.page))
	req, err := http.NewRequest("POST", l.jsonURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	b.setCookies(l.root, req, []string{"page"})
	b.finalize(req)
	pg.body, err = b.request(req)
	if err != nil {
		return nil, err
	}
	return pg, nil
}
