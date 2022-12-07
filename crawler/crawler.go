package crawler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	From          string
	Proxy         string
	MaxGoroutines int
	Logging       bool
	Output string
	// Filter string ?
	// Connect timeout ?
}

type Crawler struct {
	jobs      *channels.UnboundedChannel
	ls        *LinksStorage
	semaphore chan bool
	output string
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

	c.output = s.Output

	return c
}

func (c *Crawler) Start() {
	for {
		c.semaphore <- true
		link := c.jobs.Dequeue().(string)

		go func() {
			log.Printf("Start worker with %s", link)

			results, err := c.Worker(link)

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

func writeFile(filename string, text string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	file.WriteString(text + "\n")
}

func (c *Crawler) Worker(link string) ([]string, error) {
	if _, ok := c.ls.visited.Load(link); ok {
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

	defer func() {
		res.Body.Close()
		c.ls.visited.Store(link, struct{}{})

		u, _ := url.Parse(link)
		if _, ok := c.ls.results.Load(u.Host); !ok {
			c.ls.results.Store(u.Host, struct{}{})
			// outputs result
			fmt.Println(u.Host)
			if c.output != "" {
				writeFile(c.output, u.Host)
			}
		}
	}()
	results := findLinks(res)
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

func findLinks(responce *http.Response) []string {
	var links []string
	token := html.NewTokenizer(responce.Body)
	for {
		token_type := token.Next()

		if token_type == html.ErrorToken {
			log.Printf("html.ErrorToken")
			break
		}

		if token_type == html.StartTagToken || token_type == html.EndTagToken {
			token := token.Token()

			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						if strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://") {
							links = append(links, attr.Val)
						}
					}
				}
			}
		}
	}
	return links
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
