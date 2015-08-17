package varys

import (
	"bytes"
	"io"
	"os"

	"github.com/Xuyuanp/common"
	"github.com/Xuyuanp/logo"
)

// Options crawler options.
type Options struct {
	maxDepth int
}

// Crawler struct
type Crawler struct {
	options Options

	fetcher Fetcher
	spiders []Spider
	queue   Queue
	Logger  logo.Logger

	chURLs chan string

	wrapper common.WaitGroupWrapper
}

// NewCrawler creates a new instance of Crawler.
func NewCrawler(opts Options) (*Crawler, error) {
	return &Crawler{
		options: opts,
		fetcher: newURLFetcher(),
		queue:   NewRedisQueue(),
		Logger:  logo.New(logo.LevelDebug, os.Stdout, "[Varys] ", logo.LfullFlags),
		chURLs:  make(chan string, 1),
	}, nil
}

// Crawl starts crawling with these start URLs.
func (c *Crawler) Crawl(startURLs ...string) error {
	c.queue.Enqueue(startURLs...)
	return c.crawl()
}

func (c *Crawler) crawl() error {
	done := false
	for !done {
		select {
		case url := <-c.chURLs:
			c.crawlPage(url)
		default:
			url, err := c.queue.Dequeue()
			if err != nil || url == "" {
				c.Logger.Info("done")
				done = true
			}
			c.chURLs <- url
		}
	}
	c.queue.Cleanup()
	if failedURLs := c.queue.FailedURLs(); len(failedURLs) > 0 {
		c.Logger.Warning("failed URLs: %v", failedURLs)
	}
	return nil
}

// RegisterSpider registers spider and its middlewares.
func (c *Crawler) RegisterSpider(spider Spider, ms ...SpiderMiddleware) {
	c.spiders = append(c.spiders, ReduceSpideMiddlewares(spider, ms...))
}

func (c *Crawler) runSpider(spider Spider, url string, r io.Reader) {
	c.wrapper.Wrap(func() {
		urls, err := spider.Parse(url, r)
		if err != nil {
			c.queue.RetryURL(url)
		} else {
			c.queue.DoneURL(url)
			c.queue.Enqueue(urls...)
		}
	})
}

func (c *Crawler) crawlPage(url string) {
	c.Logger.Info("crawling page %s", url)
	body, err := c.fetcher.Fetch(url)
	if err != nil {
		c.Logger.Warning("fetch page %s failed: %s", url, err)
		c.queue.RetryURL(url)
		return
	}

	for _, spider := range c.spiders {
		c.runSpider(spider, url, bytes.NewReader(body))
	}
	c.wrapper.Wait()
}
