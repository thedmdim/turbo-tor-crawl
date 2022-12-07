package main

import (
	"flag"
	"fmt"
	"turbo-tor-crawl/crawler"
)

var (
	port   = flag.String("port", "9150", "specify the proxy port")
	target = flag.String("url", "", "specify the entry point for the crawler as a URL")
	thread = flag.Int("thread", 50, "specify the number of threads")
	output = flag.String("output", "result.txt", "specify the file to record the results of the program")
)

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintln(w, "Usage of Turbo-Tor-Crawler:")
		flag.PrintDefaults()
	}
	flag.Parse()

	fmt.Printf(
		"URL: %s\nPort: %s\nThreads: %d\n",
		*target, *port, *thread,
	)

	s := crawler.Settings{
		From:          *target,
		Proxy:         "socks5://127.0.0.1:" + *port,
		MaxGoroutines: *thread,
		Output: *output,
		//Logging: true,
	}

	c := crawler.NewCrawler(s)
	c.Start()
}
