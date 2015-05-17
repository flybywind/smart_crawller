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
				ChildSelector: "div.page_wrap a",
				TurnPage:      `vpageId=(\d+)`,
				ChildNode: func() spider.SiteNode {
					return spider.SiteNode{
						IsContainer:   true,
						ChildSelector: "div.watefall_pin.PinImage img[src][jpeg]",
						ChildNode:     nil,
						InfoText:      "div.pin_subtag_t span.fr.tag_link",
					}
				},
			}
		},
	}

	poco := spider.SiteNode{
		Url:           "http://photo.poco.cn/like/",
		ChildSelector: "div ul.item-nav.f-tdn.fl li>a.click",
		ChildNode: func() spider.SiteNode {
			return spider.SiteNode{
				ChildSelector: "div.page.f-tdn.tc a.border-color3[href^=index]",
				TurnPage:      `index-upi-p-(\d+)`,
				SiblingNum:    50,
				ChildNode: func() spider.SiteNode {
					return spider.SiteNode{
						IsContainer:   true,
						ChildSelector: "ul.listPic div.img-box img[src][jpeg]",
						ChildNode:     nil,
						InfoText:      "div ul.item-nav.f-tdn.fl li.cur>a.click",
					}
				},
			}
		},
	}

	target := []spider.SiteNode{digu, poco}
	spd := spider.NewSpider(target, 3*len(target), "/Users/flybywind/Downloads/tmp")
	spd.Run()
}
