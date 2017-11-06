package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	//"net/http/httputil"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.75 Safari/537.36"

type matcher struct {
	words [][]byte
}

type browser struct {
	client *http.Client
	jar    http.CookieJar
	debug  bool
}

type lomb24 struct {
	startURL, jsonURL string
	path              string
	pages             int
	root              *url.URL
}

func (m *matcher) match(data []byte) bool {
	var nmatch int
	for i := range m.words {
		if bytes.Contains(data, m.words[i]) {
			nmatch++
		}
	}
	return nmatch == len(m.words)
}

func (l *lomb24) match(body []byte, m *matcher) {
	if m.match(body) {
		fmt.Errorf("lombard24.ee page")
	}
}

func (l *lomb24) countPages(body []byte) (int, error) {
	var pages int
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("cannot use gquery: %v", err)
	}
	doc.Find(l.path).Each(func(i int, s *goquery.Selection) {
		pages++
	})
	return pages, nil
}

func (l *lomb24) prepare(b *browser) error {
	var err error
	l.root, err = url.Parse(l.startURL)
	if err != nil {
		return err
	}
	body, err := b.get(l.startURL)
	if err != nil {
		return err
	}
	l.pages, err = l.countPages(body)
	return err
}

func (l *lomb24) visit(page string, b *browser) ([]byte, error) {
	data := url.Values{}
	data.Set("page", page)
	req, err := http.NewRequest("POST", l.jsonURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Accept-Language", "en-US,en;q=0.9,et;q=0.8")
	req.Header.Add("User-Agent", userAgent)
	b.setCookies(l.root, req, []string{"page"})
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := readBody(resp)
	return body, err
}

func (l *lomb24) crawl(b *browser, m *matcher) {
	for i := 1; i < l.pages; i++ {
		body, err := l.visit(fmt.Sprintf("%d", i), b)
		if err != nil {
			log.Fatalf("error in page %d: %v", i, err)
		}
		if m.match(body) {
			log.Printf("page %d matches", i)
		}
	}
}

func inList(a string, list []string) bool {
	for _, s := range list {
		if a == s {
			return true
		}
	}
	return false
}

func (b *browser) setCookies(root *url.URL, req *http.Request, skip []string) {
	cks := b.jar.Cookies(root)
	for _, c := range cks {
		if inList(c.Name, skip) {
			continue
		}
		req.AddCookie(c)
	}
}

func (b *browser) get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %v", err)
	}
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	req.Header.Add("Accept-Language", "en-US,en;q=0.9,et;q=0.8")
	req.Header.Add("User-Agent", userAgent)
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot perform GET request: %v", err)
	}
	defer resp.Body.Close()
	body, err := readBody(resp)
	if err != nil {
		return nil, fmt.Errorf("cannot read body: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code was %s", resp.Status)
	}
	return body, nil
}

func readBody(resp *http.Response) ([]byte, error) {
	var (
		r   io.Reader
		err error
	)
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		r, err = gzip.NewReader(resp.Body)
	default:
		r = resp.Body
	}
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("cannot read all: %v", err)
	}
	return body, nil
}

func main() {
	l := &lomb24{
		startURL: "http://lombard24.ee/",
		jsonURL:  "http://lombard24.ee/addons/234jfsdm2n_lombard/ajax/change_page.php",
		path:     "#pg a",
	}
	b := &browser{}
	b.jar, _ = cookiejar.New(nil)
	b.client = &http.Client{
		Jar: b.jar,
	}
	if err := l.prepare(b); err != nil {
		log.Fatalf("lombard24: cannot prepare: %v", err)
	}
	words := make([][]byte, 1)
	words[0] = []byte("bottecchia")
	m := &matcher{words: words}
	l.crawl(b, m)
}
