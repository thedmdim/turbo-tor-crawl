# 🌐 turbo-tor-crawl

## What is it?
This is crawler, which collects urls of hosts which are online. 
The project was started like `.onion` only hosts finder, that's why there is an _tor_ in the name, but now you can scan what ever.
That has gone through the stages of redesign and complete refactoring from the [old attempt to make a crawler](https://github.com/Apanazar/tor_crawl).

## How does it work?
Crawler gets an init site, from which it collects all URLs and then goes to them to collects URLS too, it's kinda recursion.

## How to use it?

To get help `./turbo-tor-crawler -h`, example below:
```
Usage of Turbo-Tor-Crawler:
  -filter string
        Regex pattern which allows visiting URLs
  -from string
        specify the entry point for the crawler as a URL
  -output string
        specify the output file
  -proxy string
        specify the proxy url
  -thread int
        specify the number of threads (default 50)
  -v    enable verbosity mode  
```
The example of command to start crawling TOR sites:
```
./turbo-tor-crawl -v \
--from freshonifyfe4rmuh6qwpsexfhdrww7wnt5qmkoertwxmcuvm4woo4ad.onion \
--proxy socks5://127.0.0.1:9150`
```

## Go check our communities!
- [MOV 3371](https://t.me/mov3371)  
- [SUR.NETSTALKING](https://t.me/sur_NET)  
- [Горизонт Событий](https://t.me/+wO26CXIk4PFlMGEy)
