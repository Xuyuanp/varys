package varys

import (
	"container/ring"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

// Fetcher interface
type Fetcher interface {
	Fetch(url string) (body []byte, err error)
}

// URLFetcher struct
type URLFetcher struct {
	client  *http.Client
	prepare func(*http.Request)
}

// NewURLFetcher creates a new URLFetcher instance.
func NewURLFetcher(prepare func(*http.Request)) Fetcher {
	cfg := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: cfg}
	client := &http.Client{Transport: transport}
	return &URLFetcher{client: client, prepare: prepare}
}

// Fetch web page from url.
func (f *URLFetcher) Fetch(url string) (body []byte, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if f.prepare != nil {
		f.prepare(req)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// SmartURLFetcher is a Fetcher wrapper auto switches User-Agent.
type SmartURLFetcher struct {
	Fetcher
	userAgents *ring.Ring
	mu         sync.Mutex
}

// NewSmartURLFetcher creates new SmartURLFetcher instance.
func NewSmartURLFetcher(prepare func(*http.Request), userAgents ...string) Fetcher {
	f := &SmartURLFetcher{}
	if len(userAgents) == 0 {
		f.Fetcher = NewURLFetcher(prepare)
		return f
	}
	f.userAgents = ring.New(len(userAgents))
	for _, ua := range userAgents {
		f.userAgents.Value = ua
		f.userAgents = f.userAgents.Next()
	}
	f.Fetcher = NewURLFetcher(func(req *http.Request) {
		req.Header.Set("User-Agent", f.nextUserAgent())
		prepare(req)
	})
	return f
}

func (f *SmartURLFetcher) nextUserAgent() string {
	f.mu.Lock()
	ua := f.userAgents.Value.(string)
	f.userAgents = f.userAgents.Next()
	f.mu.Unlock()
	return ua
}
