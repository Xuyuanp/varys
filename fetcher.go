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

// FetcherOptions struct
type FetcherOptions struct {
	Timeout time.Duration
	Prepare func(*http.Request)
}

// NewFetcher creates a new Fetcher instance.
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
	ctx, cancel := context.WithTimeout(context.Background(), f.options.Timeout)
	defer cancel()
	return f.fetch(ctx, url)
}

func (f *URLFetcher) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	f.prepare(req)

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
