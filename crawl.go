package main

import (
	"log"
)

type site interface {
	start(*browser) (page, error)
	visit(page, *browser) (page, error)
}

type page interface {
	url() string
	pages() ([]page, error)
	contents() []byte
	site() site
}

func newWorkers(n int, c *crawler, b *browser, m *matcher) chan<- page {
	ch := make(chan page, n)
	for i := 0; i < n; i++ {
		go worker(ch, c, b, m)
	}
	return ch
}

// worker consumes pages from channel ch and parses them,
// calling back the crawler to signal completion with done().
func worker(ch <-chan page, c *crawler, b *browser, m *matcher) {
	for page := range ch {
		var err error
		s := page.site()
		page, err = s.visit(page, b)
		if err != nil {
			log.Printf("worker error: visiting page: %v", err)
			c.done(nil)
			continue
		}
		if page == nil {
			c.done(nil)
			continue
		}
		if m.match(page.contents()) {
			m.put(page.url())
		}
		ps, err := page.pages()
		if err != nil {
			log.Printf("page error: in page %s: %v", page.url(), err)
			c.done(nil)
			continue
		}
		c.done(ps)
	}
}

type entry struct {
	page page
	done bool
}

type crawler struct {
	urls     map[string]*entry
	fn       chan func() error
	fin      chan struct{}
	workers  chan<- page
	nworkers int
	nbusy    int
	hasWork  bool
	browser  *browser
}

func newCrawler(nworkers int, start page, b *browser, m *matcher) *crawler {
	c := &crawler{
		nworkers: nworkers,
		urls:     make(map[string]*entry),
		fn:       make(chan func() error),
		fin:      make(chan struct{}),
	}
	c.workers = newWorkers(nworkers, c, b, m)
	c.urls[start.url()] = &entry{page: start}
	go c.run()
	c.fn <- c.sched
	return c
}

// wait returns when the crawler has no more work to carry out.
func (c *crawler) wait() {
	<-c.fin
}

// sched schedules work to free workers until they are all
// busy or work has run out.
func (c *crawler) sched() error {
	var hasWork bool
	for url, entry := range c.urls {
		if entry.done {
			continue
		}
		hasWork = true
		c.urls[url].done = true
		c.nbusy++
		c.workers <- entry.page
		if c.nbusy >= c.nworkers {
			break
		}
	}
	c.hasWork = hasWork
	return nil
}

// done marks a worker as free and ingests the URLs
// that were extracted from a page.
//
// done must be always called after each task a worker
// performs.
//
// done can be called from other go routines
func (c *crawler) done(ps []page) {
	c.fn <- func() error {
		c.nbusy--
		if len(ps) == 0 {
			return nil
		}
		var hasWork bool
		for _, p := range ps {
			url := p.url()
			// TODO: here can check that we want to visit a page or not
			if _, ok := c.urls[url]; !ok {
				c.urls[url] = &entry{
					page: p,
				}
				hasWork = true
			}
		}
		if hasWork {
			c.hasWork = true
		}
		return nil
	}
}

// run handles all synchronized work on the crawler and
// invokes the scheduler to keep all workers busy until
// work (pages to visit) has run out.
func (c *crawler) run() {
	for fn := range c.fn {
		if err := fn(); err != nil {
			log.Printf("crawler error: %s", err)
		}
		// No more work and no results to wait for, exit.
		if !c.hasWork && c.nbusy == 0 {
			break
		}
		if c.hasWork {
			c.sched()
		}
	}
	close(c.workers)
	close(c.fin)
}

/*
func main() {
	// TODO: as real flag
	nworkers := 4
	flag.Parse()
	c, err := newCrawler(flag.Arg(0), nworkers)
	if err != nil {
		log.Fatalf("cannot start crawler: %s", err)
	}
	c.wait()
	for url := range c.urls {
		fmt.Printf("%s\n", url)
	}
}
*/
