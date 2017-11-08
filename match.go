package main

import (
	"bytes"
	"log"
)

type matcher struct {
	words [][]byte
	urls  chan string
	done  chan struct{}
}

func newMatcher(words [][]byte) *matcher {
	return &matcher{
		words: words,
		urls:  make(chan string),
		done:  make(chan struct{}),
	}
}

func (m *matcher) run() {
	for u := range m.urls {
		log.Printf("MATCH %s", u)
	}
	m.done <- struct{}{}
}

func (m *matcher) stop() {
	close(m.urls)
	<-m.done
}

func (m *matcher) put(url string) {
	m.urls <- url
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
