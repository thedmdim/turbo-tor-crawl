package main

import (
	"flag"
	"fmt"
	"turbo-tor-crawl/crawler"
)

var (
	from = flag.String("from", "", "specify the entry point for the crawler as a URL")
	proxy   = flag.String("proxy", "", "specify the proxy url")
	threads = flag.Int("thread", 50, "specify the number of threads")
	output = flag.String("output", "", "specify the output file")
	verbosity = flag.Bool("v", false, "enable verbosity mode")
	filter = flag.String("filter", "", "Regex pattern which allows visiting URLs")
)

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintln(w, "Usage of Turbo-Tor-Crawler:")
		flag.PrintDefaults()
	}
	flag.Parse()

	fmt.Printf(
		"From: %s\nPort: %s\nThreads: %d\n",
		*from, *proxy, *threads,
	)

	s := crawler.Settings{
		From:          *from,
		Proxy:         *proxy,
		Threads: *threads,
		Output: *output,
		Logging: *verbosity,
		Filter: *filter,
	}

	c := crawler.NewCrawler(s)
	c.Start()
}
