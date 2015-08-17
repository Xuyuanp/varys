package varys

import "io"

// Spider interface.
type Spider interface {
	Parse(url string, r io.Reader) ([]string, error)
}

// SpiderFunc type Spider.
type SpiderFunc func(url string, r io.Reader) ([]string, error)

// Parse implements Spider interface.
func (sf SpiderFunc) Parse(url string, r io.Reader) ([]string, error) {
	return sf(url, r)
}

// SpiderMiddleware type.
type SpiderMiddleware func(Spider) Spider
