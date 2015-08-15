package varys

import (
	"os"

	"github.com/PuerkitoBio/goquery"
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

type Options struct{}

type Crawler struct {
	options Options

	spiders []Spider
	Logger  logo.Logger

	pool *redis.Pool

	chURLs  chan string
	chEntry <-chan entry

	done chan bool

	wrapper common.WaitGroupWrapper
}

func NewCrawler(opts Options) (*Crawler, error) {
	return &Crawler{
		options: opts,
		Logger:  logo.New(logo.LevelDebug, os.Stdout, "[Varys] ", logo.LfullFlags),
		chURLs:  make(chan string),
		done:    make(chan bool),
		pool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", "127.0.0.1:6379")
			},
			Wait: true,
		},
	}, nil
}

func (c *Crawler) Crawl(startURLs ...string) error {
	c.Enqueue(startURLs...)
	return c.crawl()
}

func (c *Crawler) crawl() error {
	c.chEntry = NewDownloader(4, c.chURLs)
	c.wrapper.Wrap(func() {
		for {
			url, err := c.Dequeue()
			if err != nil || url == "" {
				c.Logger.Warning("done")
				c.done <- true
				return
			}
			c.PendingURL(url)
			c.chURLs <- url
		}
	})
	c.waitingForDownloader()
	c.wrapper.Wait()
	return nil
}

func (c *Crawler) Enqueue(urls ...string) {
	conn := c.pool.Get()
	defer conn.Close()

	c.Logger.Debug("enqueue urls: %v", urls)

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
		c.Logger.Debug("enqueue url: %s", url)
		_, err := conn.Do("SADD", queueReady, url)
		if err != nil {
			c.Logger.Warning("enqueue url %s failed: %s", url, err)
		}
	}
}

func (c *Crawler) Dequeue() (url string, err error) {
	c.Logger.Debug("dequeue")
	conn := c.pool.Get()
	defer conn.Close()
	url, err = redis.String(conn.Do("SRANDMEMBER", queueReady))
	return
}

func (c *Crawler) PendingURL(url string) error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queueReady, queuePending, url)
	return err
}

func (c *Crawler) DoneURL(url string) error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queuePending, queueDone, url)
	return err
}

func (c *Crawler) RetryURL(url string) error {
	conn := c.pool.Get()
	defer conn.Close()
	_, err := conn.Do("SMOVE", queuePending, queueFailed, url)
	return err
}

func (c *Crawler) RegisterSpider(spider Spider, ms ...SpiderMiddleware) {
	for i := len(ms) - 1; i >= 0; i-- {
		spider = ms[i](spider)
	}
	c.spiders = append(c.spiders, spider)
}

func (c *Crawler) waitingForDownloader() {
	for {
		en := <-c.chEntry
		c.Logger.Debug("got entry:%+v", en)
		c.processDocument(en)
	}
}

func (c *Crawler) processDocument(en entry) {
	for _, spider := range c.spiders {
		c.wrapper.Wrap(func() {
			c.runSpider(spider, en.url, goquery.CloneDocument(en.doc))
		})
	}
}

func (c *Crawler) runSpider(spider Spider, url string, doc *goquery.Document) {
	urls, err := spider.Parse(doc)
	if err != nil {
		c.RetryURL(url)
	} else {
		c.DoneURL(url)
		c.Enqueue(urls...)
	}
}
