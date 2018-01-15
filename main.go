package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/antchfx/antch"
	"github.com/antchfx/htmlquery"
)

type item struct {
	Title string `json:"title"`
	Link  string `json:"link"`
	Desc  string `json:"desc"`
}

type trimSpacePipeline struct {
	next antch.PipelineHandler
}

func (p *trimSpacePipeline) ServePipeline(v antch.Item) {
	vv := v.(*item)
	vv.Title = strings.TrimSpace(vv.Title)
	vv.Desc = strings.TrimSpace(vv.Desc)
	p.next.ServePipeline(vv)
}

func newTrimSpacePipeline() antch.Pipeline {
	return func(next antch.PipelineHandler) antch.PipelineHandler {
		return &trimSpacePipeline{next}
	}
}

type jsonOutputPipeline struct{}

func (p *jsonOutputPipeline) ServePipeline(v antch.Item) {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	std := os.Stdout
	std.Write(b)
	std.Write([]byte("\n"))
}

func newJsonOutputPipeline() antch.Pipeline {
	return func(next antch.PipelineHandler) antch.PipelineHandler {
		return &jsonOutputPipeline{}
	}
}

type dmozSpider struct{}

func (s *dmozSpider) ServeSpider(c chan<- antch.Item, res *http.Response) {
	doc, err := antch.ParseHTML(res)
	if err != nil {
		panic(err)
	}
	for _, node := range htmlquery.Find(doc, "//div[@id='site-list-content']/div") {
		v := new(item)
		v.Title = htmlquery.InnerText(htmlquery.FindOne(node, "//div[@class='site-title']"))
		v.Link = htmlquery.SelectAttr(htmlquery.FindOne(node, "//a"), "href")
		v.Desc = htmlquery.InnerText(htmlquery.FindOne(node, "//div[contains(@class,'site-descr')]"))
		c <- v
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	c := make(chan struct{})
	crawler := &antch.Crawler{Exit: c}
	crawler.UseCompression()

	crawler.Handle("dmoztools.net", &dmozSpider{})
	crawler.UsePipeline(newTrimSpacePipeline(), newJsonOutputPipeline())

	startURLs := []string{
		"http://dmoztools.net/Computers/Programming/Languages/Python/Books/",
		"http://dmoztools.net/Computers/Programming/Languages/Python/Resources/",
	}

	go func() {
		crawler.StartURLs(startURLs)
		<-sigs // `CTRL-C` to stop crawler.
		close(c)
	}()
	// crawler is block waiting for a signal.
	<-crawler.Exit
	fmt.Println("exiting crawler")
}
