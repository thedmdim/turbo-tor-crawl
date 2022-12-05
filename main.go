package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/alimate/unbounded-channel/channels"
	"golang.org/x/net/html"
)

var (
	proxy  = flag.Bool("proxy", true, "specify you will use a proxy")
	port   = flag.String("port", "9150", "specify the proxy port")
	target = flag.String("url", "", "specify the entry point for the crawler as a URL")
)

func worker(jobs *channels.UnboundedChannel, hosts *sync.Map, visited *sync.Map) {
	origin := jobs.Dequeue().(string)

	if _, ok := visited.Load(origin); ok {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			fmt.Print()
		}
	}()
	res, _ := http.Get(origin)

	defer func() {
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
		log.Println(err)
	}

	links := []string{}
	findLinks(&links, dom)
	relLinksToAbs(&links, origin)

	for _, link := range links {
		if _, ok := visited.Load(link); !ok {
			jobs.Enqueue(link)
		}
	}

}

func findLinks(links *[]string, n *html.Node) {
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

		abs.RawQuery = ""
		if flink := abs.String(); link != flink {
			(*links)[i] = flink
		}

	}

}

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintln(w, "Usage of Turbo-Tor-Crawler:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *proxy {
		proxy_url, err := url.Parse("socks5://127.0.0.1:" + *port)
		if err != nil {
			log.Println(err)
		}

		http.DefaultTransport = &http.Transport{
			Proxy: http.ProxyURL(proxy_url),
		}
	}

	jobs := channels.NewUnboundedChannel()
	results := new(sync.Map)
	visited := new(sync.Map)

	jobs.Enqueue(*target)
	semaphore := make(chan bool, 6)

	for {
		semaphore <- true
		go func() {
			worker(jobs, results, visited)
			<-semaphore
		}()
	}
}
