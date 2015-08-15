package varys

import "github.com/PuerkitoBio/goquery"

type Spider interface {
	Parse(*goquery.Document) ([]string, error)
}

type SpiderFunc func(*goquery.Document) ([]string, error)

func (sf SpiderFunc) Parse(doc *goquery.Document) ([]string, error) {
	return sf(doc)
}

type SpiderMiddleware func(Spider) Spider

func NewHostMiddleware(host string) SpiderMiddleware {
	return func(spider Spider) Spider {
		return SpiderFunc(func(doc *goquery.Document) ([]string, error) {
			if doc.Url.Host != host {
				return nil, nil
			}
			return spider.Parse(doc)
		})
	}
}
