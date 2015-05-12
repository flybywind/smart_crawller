package main

import (
	"smart_crawller/spider"
)

func main() {
	digu := spider.SiteNode{
		Url:          "http://www.digu.com/",
		NextSelector: "div.pin.index_cate a.png24.tag_link",
		NextNode: func() spider.SiteNode {
			return spider.SiteNode{
				IsContainer:  true,
				NextSelector: "div.watefall_pin.PinImage img[src][jpeg]",
				NextNode:     nil,
				InfoText:     "div.pin_subtag_t>span.fr.tag_link",
			}
		},
	}
	target := []spider.SiteNode{digu}
	spd := spider.NewSpider(target, 1, "/Users/flybywind/Downloads/tmp")
	spd.Run()
}
