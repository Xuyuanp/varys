package varys

import "io"

// Spider interface.
type Spider interface {
	Parse(crawler *Crawler, url string, r io.Reader, err error) ([]string, error)
}

// SpiderFunc type Spider.
type SpiderFunc func(crawler *Crawler, url string, r io.Reader, err error) ([]string, error)

// Parse implements Spider interface.
func (sf SpiderFunc) Parse(crawler *Crawler, url string, r io.Reader, err error) ([]string, error) {
	return sf(crawler, url, r, err)
}

// SpiderMiddleware type.
type SpiderMiddleware func(Spider) Spider

// ReduceSpideMiddlewares merges multi SpiderMiddlewares and a spider into a new Spider.
func ReduceSpideMiddlewares(spider Spider, ms ...SpiderMiddleware) Spider {
	for i := len(ms) - 1; i >= 0; i-- {
		spider = ms[i](spider)
	}
	return spider
}
