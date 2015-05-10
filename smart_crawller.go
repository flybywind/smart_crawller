package main

import (
	"fmt"
	"smart_crawller/spider"
)

func main() {
	fmt.Println(spider.GetFullNormalizeUrl("http://abc.com", ""))
	fmt.Println(spider.GetFullNormalizeUrl("http://abc.com", "http://xyz.com"))
	fmt.Println(spider.GetFullNormalizeUrl("http://abc.com", "/ABC"))
	fmt.Println(spider.GetFullNormalizeUrl("HTtp://abc.com", "/ABC"))
	fmt.Println(spider.GetFullNormalizeUrl("http://abc.com/a/b/x", "ABC"))
	fmt.Println(spider.GetFullNormalizeUrl("http://abc.com/a/b/x", "ABc"))
}
