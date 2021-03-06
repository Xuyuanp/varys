/*
 * Copyright 2015 Xuyuan Pang
 * Author: Xuyuan Pang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package varys

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/Xuyuanp/logo"
)

// Error var
var (
	ErrRunning = errors.New("already running")
)

// Options crawler options.
type Options struct {
	MaxDepth int
	SleptMin int
	SleptMax int
}

// Crawler struct
type Crawler struct {
	options Options

	fetcher Fetcher
	spiders []Spider
	queue   Queue
	Logger  logo.Logger

	running bool

	chURLs chan string

	wrapper Wrapper
}

var random = rand.New(rand.NewSource(time.Now().Unix()))

// NewCrawler creates a new instance of Crawler.
func NewCrawler(opts Options, queue Queue, fetcher Fetcher) (*Crawler, error) {
	return &Crawler{
		options: opts,
		fetcher: fetcher,
		queue:   queue,
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
	if c.running {
		return ErrRunning
	}
	c.running = true
	c.queue.Repaire()
	done := false
	for !done {
		select {
		case url := <-c.chURLs:
			c.crawlPage(url)
			c.sleep()
		default:
			url, err := c.queue.Dequeue()
			if err != nil || url == "" {
				c.Logger.Info("done")
				done = true
			} else {
				c.chURLs <- url
			}
		}
	}
	c.queue.Cleanup()
	if failedURLs := c.queue.FailedURLs(); len(failedURLs) > 0 {
		c.Logger.Warning("failed URLs: %v", failedURLs)
	}
	c.running = false
	return nil
}

// Running returns crawler's running status.
func (c *Crawler) Running() bool {
	return c.running
}

func (c *Crawler) sleep() {
	rang := c.options.SleptMax - c.options.SleptMin
	if rang <= 0 {
		return
	}
	r := random.Intn(rang)
	dur := c.options.SleptMin + r
	time.Sleep(time.Second * time.Duration(dur))
}

// RegisterSpider registers spider and its middlewares.
func (c *Crawler) RegisterSpider(spider Spider, ms ...SpiderMiddleware) {
	c.spiders = append(c.spiders, ReduceSpideMiddlewares(spider, ms...))
}

func (c *Crawler) runSpider(spider Spider, url string, r io.Reader, err error) {
	c.wrapper.Wrap(func() {
		urls, err := spider.Parse(c, url, r, err)
		if err != nil {
			c.Logger.Warning("fetch url %s failed: %s", url, err)
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

	for _, spider := range c.spiders {
		c.runSpider(spider, url, bytes.NewReader(body), err)
	}
	c.wrapper.Wait()
}
