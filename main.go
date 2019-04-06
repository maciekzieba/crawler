package main

import (
	"fmt"
	"sync"
)

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(wg *sync.WaitGroup, url string, depth int, fetcher Fetcher, output chan string) {
	defer wg.Done()
	// TODO: Fetch URLs in parallel.
	if depth <= 0 {
		return
	}

	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		output <- err.Error()
		return
	}

	output <- fmt.Sprintf("found: %s %q", url, body)
	for _, u := range urls {
		wg.Add(1)
		go Crawl(wg, u, depth-1, fetcher, output)
	}
	return
}

func main() {
	output := make(chan string)
	var wg sync.WaitGroup

	cacheFetcher := NewCacheFetcher(fetcher)
	wg.Add(1)
	go Crawl(&wg, "https://golang.org/", 4, &cacheFetcher, output)
	go func() {
		for message := range output {
			fmt.Printf("%s\n", message)
		}
	}()

	wg.Wait()
}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}

type CacheItem struct {
	body string
	urls []string
}

type CacheFetcher struct {
	items   map[string]CacheItem
	mux     sync.Mutex
	fetcher Fetcher
}

func (f *CacheFetcher) Fetch(url string) (string, []string, error) {
	f.mux.Lock()
	item, cacheExists := f.items[url]
	f.mux.Unlock()
	if cacheExists {
		fmt.Printf("hit from cache: %s %s\n", url, item.body)
		return item.body, item.urls, nil
	} else {
		body, urls, err := f.fetcher.Fetch(url)
		if err == nil {
			f.mux.Lock()
			f.items[url] = CacheItem{body, urls}
			f.mux.Unlock()
		}
		return body, urls, err
	}
}

func NewCacheFetcher(fetcher Fetcher) CacheFetcher {
	return CacheFetcher{
		items:   make(map[string]CacheItem),
		fetcher: fetcher,
	}
}
