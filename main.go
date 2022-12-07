package main

import (
	"turbo-tor-crawl/crawler"
)

func main() {
	c := crawler.Crawler{
		From: "http://freshonifyfe4rmuh6qwpsexfhdrww7wnt5qmkoertwxmcuvm4woo4ad.onion/",
		Proxy: "socks5://127.0.0.1:9150",
		MaxGoroutines: 10,
		//Logging: true,
	}

	c.Init()

	c.Start()
}
