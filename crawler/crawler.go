package crawler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/alimate/unbounded-channel/channels"
	
	
	"golang.org/x/net/html"
)

type Settings struct {
	From          string
	Proxy         string
	Threads int
	Logging       bool
	Output string
	Filter string
	// Connect timeout ?
}

type linksStorage struct {
	// literally results of crawling
	results sync.Map
	// unique constraint to not request same link twice
	visited sync.Map
}

type crawler struct {
	jobs      *channels.UnboundedChannel
	ls        *linksStorage
	semaphore chan bool
	output string
	filter string
}

func NewCrawler(s Settings) *crawler {
	
	// output
	// Enable writing output to file 
	log.SetOutput(io.Discard)
	if s.Logging {
		log.SetOutput(os.Stdout)
	}

	// proxy
	// Set proxy for all requests
	if s.Proxy != "" {
		proxy, err := url.Parse(s.Proxy)
		if err != nil {
			log.Panicf("Can't read proxy address: %e", err)

		}
		http.DefaultTransport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}

	// threads
	// Set number of threads requesting URL at the same time
	if s.Threads == 0 {
		log.Printf("No max goroutine provided, default %d", s.Threads)
	}

	// from
	// A site from which crawling starts
	if s.From == "" {
		log.Fatalf("Provide url from which we'll start crawling")
	}

	u, err := url.Parse(s.From)
	if err != nil {
		log.Fatalf("Cannot parse URL of %s", s.From)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}

	// Create and pass settings to crawl object
	c := new(crawler)

	c.jobs = channels.NewUnboundedChannel()
	c.jobs.Enqueue(u.String())

	c.ls = new(linksStorage)
	c.semaphore = make(chan bool, s.Threads)

	c.output = s.Output
	c.filter = s.Filter

	return c
}

func (c *crawler) Start() {
	// Start crawling
	for {
		c.semaphore <- true
		link := c.jobs.Dequeue().(string)

		go func() {
			log.Printf("Start worker with %s", link)

			results, err := c.worker(link)

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

func (c *crawler) worker(link string) ([]string, error) {
	// A worker which requests site and collects links from it

	// Check if we have already requsted the url
	if _, ok := c.ls.visited.Load(link); ok {
		return nil, fmt.Errorf("%s is already visited", link)
	}

	// Check if url matches pattern
	matched, err := regexp.MatchString(c.filter, link)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("%s doesn't match filter", link)
	}

	// Request URL
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
			// Outputs result
			fmt.Println(u.Host)
			if c.output != "" {
				writeFile(c.output, u.Host)
			}
		}
	}()

	// Find all links and make them absolute
	results := findLinks(res.Body)
	relLinksToAbs(&results, link)

	return results, nil

}

func findLinks(body io.Reader) []string {
	// Find all links in HTML

	var links []string
	token := html.NewTokenizer(body)
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
	// Make all links absolute

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

func writeFile(filename string, text string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	file.WriteString(text + "\n")
}