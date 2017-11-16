package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.75 Safari/537.36"

type browser struct {
	client *http.Client
	jar    http.CookieJar
}

func newBrowser() *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{
		jar: jar,
		client: &http.Client{
			Jar: jar,
		},
	}
}

func (b *browser) finalize(req *http.Request) {
	req.Header.Add("Accept-Language", "en-US,en;q=0.9,et;q=0.8")
	req.Header.Add("User-Agent", userAgent)
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
	b.finalize(req)
	return b.request(req)
}

func (b *browser) request(req *http.Request) ([]byte, error) {
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot perform GET request: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read body: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code was %s", resp.Status)
	}
	return body, nil
}

func (b *browser) normalize(base *url.URL, surl string) (string, error) {
	u, err := url.Parse(surl)
	if err != nil {
		return "", err
	}
	// Ignore links to other domains
	// TODO: be more lax about 80 and 443 with right scheme
	if u.Host != "" && u.Host != base.Host {
		return "", nil
	}
	u.Host = base.Host
	if u.Scheme != "" {
		// Skip unhandled schemes
		if u.Scheme != "http" || u.Scheme != "https" {
			return "", nil
		}
		if u.Scheme != base.Scheme {
			return "", fmt.Errorf("schema is %s, it was %s", u.Scheme, base.Scheme)
		}
	}
	u.Scheme = base.Scheme
	// Opaque: ignored
	// User: ignored
	if u.Path[0] != '/' {
		u.Path = base.Path + u.Path
	}
	u.Path = path.Clean(u.Path)
	if u.Path == "/" {
		u.Path = ""
	}
	u.Fragment = ""
	surl = u.String()
	// Skip internal link, only fragment
	if surl[0] == '#' {
		return "", nil
	}
	return surl, nil
}

func inList(a string, list []string) bool {
	for _, s := range list {
		if a == s {
			return true
		}
	}
	return false
}
