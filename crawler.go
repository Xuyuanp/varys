package varys

import (
	"bytes"
	"io"
	"os"

	"github.com/Xuyuanp/common"
	"github.com/Xuyuanp/logo"
	"github.com/garyburd/redigo/redis"
)

const (
	queueReady   = "queue-ready"
	queuePending = "queue-pending"
	queueDone    = "queue-done"
	queueFailed  = "queue-failed"
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
	Logger  logo.Logger

	pool *redis.Pool

	chURLs chan string

	wrapper common.WaitGroupWrapper
}

// NewCrawler creates a new instance of Crawler.
func NewCrawler(opts Options) (*Crawler, error) {
	return &Crawler{
		options: opts,
		fetcher: newURLFetcher(),
		Logger:  logo.New(logo.LevelDebug, os.Stdout, "[Varys] ", logo.LfullFlags),
		chURLs:  make(chan string, 1),
		pool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", "127.0.0.1:6379")
			},
			Wait: true,
		},
	}, nil
}

// Crawl starts crawling with these start URLs.
func (c *Crawler) Crawl(startURLs ...string) error {
	c.enqueue(startURLs...)
	return c.crawl()
}

func (c *Crawler) crawl() error {
	done := false
	for !done {
		select {
		case url := <-c.chURLs:
			c.crawlPage(url)
		default:
			url, err := c.dequeue()
			if err != nil || url == "" {
				c.Logger.Info("done")
				done = true
			}
			c.pendingURL(url)
			c.chURLs <- url
		}
	}
	c.cleanup()
	return nil
}

func (c *Crawler) enqueue(urls ...string) {
	conn := c.pool.Get()
	defer conn.Close()

	for _, url := range urls {
		if dup, err := redis.Bool(conn.Do("SISMEMBER", queueFailed, url)); err == nil && dup {
			continue
		}
		if dup, err := redis.Bool(conn.Do("SISMEMBER", queueDone, url)); err == nil && dup {
			continue
		}
		if dup, err := redis.Bool(conn.Do("SISMEMBER", queuePending, url)); err == nil && dup {
			continue
		}
		_, err := conn.Do("SADD", queueReady, url)
		if err != nil {
			c.Logger.Warning("enqueue url %s failed: %s", url, err)
		}
	}
}

func (c *Crawler) dequeue() (url string, err error) {
	conn := c.pool.Get()
	defer conn.Close()
	url, err = redis.String(conn.Do("SRANDMEMBER", queueReady))
	return
}

func (c *Crawler) repaire() {
	conn := c.pool.Get()
	defer conn.Close()
	conn.Do("SUNION", queueReady)
}

func (c *Crawler) pendingURL(url string) error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queueReady, queuePending, url)
	return err
}

func (c *Crawler) doneURL(url string) error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queuePending, queueDone, url)
	return err
}

func (c *Crawler) retryURL(url string) error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queuePending, queueFailed, url)
	return err
}

// RegisterSpider registers spider and its middlewares.
func (c *Crawler) RegisterSpider(spider Spider, ms ...SpiderMiddleware) {
	for i := len(ms) - 1; i >= 0; i-- {
		spider = ms[i](spider)
	}
	c.spiders = append(c.spiders, spider)
}

func (c *Crawler) runSpider(spider Spider, url string, r io.Reader) {
	c.wrapper.Wrap(func() {
		urls, err := spider.Parse(url, r)
		if err != nil {
			c.retryURL(url)
		} else {
			c.doneURL(url)
			c.enqueue(urls...)
		}
	})
}

func (c *Crawler) crawlPage(url string) {
	c.Logger.Info("crawling page %s", url)
	body, err := c.fetcher.Fetch(url)
	if err != nil {
		c.Logger.Warning("fetch page %s failed: %s", url, err)
		c.retryURL(url)
		return
	}

	for _, spider := range c.spiders {
		c.runSpider(spider, url, bytes.NewReader(body))
	}
	c.wrapper.Wait()
}

func (c *Crawler) cleanup() {
	// conn := c.pool.Get()
	// defer conn.Close()
	//
	// conn.Do("DEL", queueDone)
	// conn.Do("DEL", queueReady)
	// conn.Do("DEL", queuePending)
}
