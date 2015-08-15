package varys

import (
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/Xuyuanp/logo"
)

type entry struct {
	url string
	doc *goquery.Document
}

func NewDownloader(taskNum int, chURL <-chan string) <-chan entry {
	if taskNum < 0 {
		taskNum = 4
	}
	chEntry := make(chan entry)
	for i := 0; i < taskNum; i++ {
		go func() {
			for {
				url := <-chURL
				resp, err := http.Get(url)
				if err != nil {
					logo.Warning("get page %s failed: %s", url, err)
				}
				doc, err := goquery.NewDocumentFromResponse(resp)
				if err != nil {

				}
				chEntry <- entry{url: url, doc: doc}
			}
		}()
	}
	return chEntry
}
