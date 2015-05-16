package main

import (
	"smart_crawller/spider"
)

func main() {
	digu := spider.SiteNode{
		Url:           "http://www.digu.com/",
		ChildSelector: "div.pin.index_cate a.png24.tag_link",
		ChildNode: func() spider.SiteNode {
			return spider.SiteNode{
				IsContainer:   true,
				ChildSelector: "div.watefall_pin.PinImage img[src][jpeg]",
				ChildNode:     nil,
				InfoText:      "div.pin_subtag_t span.fr.tag_link",
				TurnPage:      "?vpageId=",
				ContainerId:   0,
			}
		},
	}

	/*poco := spider.SiteNode{
		Url:           "http://photo.poco.cn/like/",
		ChildSelector: "div ul.item-nav.f-tdn.fl li>a.click",
		ChildNode: func() spider.SiteNode {
			return spider.SiteNode{
				IsContainer:    true,
				ChildSelector:  "ul.listPic div.img-box img[src][jpeg]",
				ChildNode:      nil,
				InfoText:       "div ul.item-nav.f-tdn.fl li>a.click",
				NextSiblingSel: "a#nextpage",
			}
		},
	}*/

	target := []spider.SiteNode{digu}
	spd := spider.NewSpider(target, len(target), "/Users/flybywind/Downloads/tmp")
	spd.Run()
}
