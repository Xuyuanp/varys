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

// ReduceSpideMiddlewares merges multi SpiderMiddlewares and a spider into a new Spider.
func ReduceSpideMiddlewares(spider Spider, ms ...SpiderMiddleware) Spider {
	for i := len(ms) - 1; i >= 0; i-- {
		spider = ms[i](spider)
	}
	return spider
}
