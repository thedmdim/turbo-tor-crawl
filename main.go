package main

import (
	//"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	//"time"

	"github.com/alimate/unbounded-channel/channels"
	"golang.org/x/net/html"
)

func main() {
	// set global proxy
	proxy, err := url.Parse("socks5://127.0.0.1:9150")
	if err != nil {
		panic(err)
	}
	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}

	jobs := channels.NewUnboundedChannel()
	// var hosts sync.Map
	// var visited sync.Map
	results := new(sync.Map)
	visited := new(sync.Map)

	jobs.Enqueue("http://freshonifyfe4rmuh6qwpsexfhdrww7wnt5qmkoertwxmcuvm4woo4ad.onion")

	semaphore := make(chan bool, 6)

	for {
		semaphore <- true
		go func() {
			worker(jobs, results, visited)
			<-semaphore
		}()
	}
}

func worker (jobs *channels.UnboundedChannel, hosts *sync.Map, visited *sync.Map) {

	origin := jobs.Dequeue().(string)

	if _, ok := visited.Load(origin); ok {
		// check if we have already requsted the url
		return
	}

	res, err := http.Get(origin)
	if err != nil {
		log.Println(err)
		return
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("%s returned %d", origin, res.StatusCode)
		return
	}

	defer func(){
		res.Body.Close()
		visited.Store(origin, struct{}{})

		u, _ := url.Parse(origin)
		if _, ok := hosts.Load(u.Host); !ok {
			hosts.Store(u.Host, struct{}{})
			fmt.Println(u.Host)
		}
	}()


	dom, err := html.Parse(res.Body)
	if err != nil {
		return
	}

	var links []string
	findLinks(&links, dom)
	relLinksToAbs(&links, origin)
	

	for _, link := range links {
		if _, ok := visited.Load(link); !ok {
			jobs.Enqueue(link)
		}
	}
	
}

func findLinks(links *[]string, n *html.Node) {
	// Рекурсивно ищет a[href] на странице
	if n == nil {
		return
	}

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				*links = append(*links, attr.Val)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findLinks(links, c)
	}
	
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



