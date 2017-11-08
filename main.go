package main

import (
	"flag"
	"log"
)

func main() {
	nworkers := 4
	flag.Parse()
	words := make([][]byte, 0)
	for _, f := range flag.Args() {
		words = append(words, []byte(f))
	}
	if len(words) == 0 {
		log.Fatal("specify some words to search")
	}
	m := newMatcher(words)
	go m.run()
	b := newBrowser()
	l, err := newLomb24()
	start, err := l.start(b)
	if err != nil {
		log.Fatalf("cannot start lomb24: %v", err)
	}
	cr := newCrawler(nworkers, start, b, m)
	cr.wait()
	m.stop()
}
