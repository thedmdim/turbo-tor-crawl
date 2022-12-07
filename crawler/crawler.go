package crawler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/alimate/unbounded-channel/channels"
	"golang.org/x/net/html"
)

type LinksStorage struct {
	// literally results of crawling
	results sync.Map
	// unique constraint to not request same link twice
	visited sync.Map
}

type Settings struct {
	From string
	Proxy string
	MaxGoroutines int
	Logging bool
	// Filter string ?
	// Connect timeout ?
}

type Crawler struct {
	jobs *channels.UnboundedChannel
	ls *LinksStorage
	semaphore chan bool
}


func NewCrawler(s Settings) *Crawler {
	// Проверка переданных в Crawler параметров

	if s.Logging {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(io.Discard)
	}

	if s.Proxy != "" {
		proxy, err := url.Parse(s.Proxy)
		if err != nil {
			log.Panicf("Can't read proxy address: %e", err)

		}
		http.DefaultTransport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}

	semaphoreDefault := 6
	if s.MaxGoroutines == 0 {
		log.Printf("No max goroutine provided, default %d", semaphoreDefault)
		s.MaxGoroutines = semaphoreDefault
	}

	if s.From == "" {
		log.Fatalf("Provide url from which we'll start crawling")
	}

	u, err := url.Parse(s.From)
	if err != nil {
		log.Fatalf("Cannot parse URL of %s", s.From)
	}

	c := new(Crawler)

	c.jobs = channels.NewUnboundedChannel()
	c.jobs.Enqueue(u.String())

	c.ls = new(LinksStorage)
	c.semaphore = make(chan bool, s.MaxGoroutines)

	return c
}

func (c *Crawler) Start() {
	for {
		c.semaphore <- true
		link := c.jobs.Dequeue().(string)

		go func() {
			log.Printf("Start worker with %s", link)

			results, err := Worker(link, c.ls)

			if err != nil {
				log.Printf("Worker: %v", err)
			}

			log.Printf("Found %d links from %s", len(results), link)

			for _, link := range results {
				c.jobs.Enqueue(link)
			}

			<-c.semaphore
		}()
	}
}

func Worker(link string, ls *LinksStorage) ([]string, error) {
	if _, ok := ls.visited.Load(link); ok {
		// check if we have already requsted the url
		return nil, fmt.Errorf("%s is already visited", link)
	}

	res, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", link, res.StatusCode)
	}

	defer func(){
		res.Body.Close()
		ls.visited.Store(link, struct{}{})

		u, _ := url.Parse(link)
		if _, ok := ls.results.Load(u.Host); !ok {
			ls.results.Store(u.Host, struct{}{})
			// outputs result
			fmt.Println(u.Host)
		}
	}()

	dom, err := html.Parse(res.Body)
	if err != nil {
		return nil, fmt.Errorf("can't parse HTML for %s: %w", link, err)
	}

	results, err := findLinks(dom)
	if err != nil {
		return nil, fmt.Errorf("haven't found any link at %s", link)
	}

	relLinksToAbs(&results, link)
	
	// // delete from result already visited links
	// for i:=0;i==len(results);i++{
	// 	if _, ok := ls.visited.Load(results[i]); ok {
	// 		results = append(results[:i], results[i+1:]...)
	// 		i--
	// 	}
	// }


	return results, nil

}

func findLinks(n *html.Node) ([]string, error) {

	var links []string
	// Рекурсивно ищет a[href] на странице

	var f func(n *html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
	
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return links, nil

	
}

func relLinksToAbs(links *[]string, baseURL string) {
	base, _ := url.Parse(baseURL)

	for i, link := range *links {
		abs, err := base.Parse(link)
		if err != nil {
			continue
		}

		abs.RawQuery = "" // del get params
		if flink := abs.String(); link != flink {
			(*links)[i] = flink
		}
	
	}

}
