package varys

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Fetcher interface
type Fetcher interface {
	Fetch(url string) (body []byte, err error)
}

type URLFetcher struct {
	client *http.Client
}

func NewURLFetcher() *URLFetcher {
	cfg := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: cfg}
	client := &http.Client{Transport: transport}
	return &URLFetcher{client: client}
}

// Fetch web page from url.
func (f *URLFetcher) Fetch(url string) (body []byte, err error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}
