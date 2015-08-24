package varys

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// Fetcher interface
type Fetcher interface {
	Fetch(url string) (body []byte, err error)
}

// URLFetcher struct
type URLFetcher struct {
	options FetcherOptions
	client  *http.Client
}

type FetcherOptions struct {
	Timeout time.Duration
	Prepare func(*http.Request)
}

func NewFetcher(opts FetcherOptions) Fetcher {
	cfg := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: cfg}
	client := &http.Client{Transport: transport}
	return &URLFetcher{
		client:  client,
		options: opts,
	}
}

// Fetch web page from url.
func (f *URLFetcher) Fetch(url string) (body []byte, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	f.prepare(req)
	ctx, cancel := f.newContext(context.Background())
	defer cancel()
	resp, err := ctxhttp.Do(ctx, f.client, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func (f *URLFetcher) prepare(req *http.Request) {
	if f.options.Prepare != nil {
		f.options.Prepare(req)
	}
}

func (f *URLFetcher) newContext(parent context.Context) (context.Context, context.CancelFunc) {
	if f.options.Timeout > 0 {
		return context.WithTimeout(parent, f.options.Timeout)
	}
	return context.WithCancel(parent)
}
