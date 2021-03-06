package main

import (
	"crypto/md5"
	"fmt"
	"smart_crawller/spider"
	"testing"
)

func TestSpiderHistoryMem(t *testing.T) {
	sm := spider.SpiderHistoryMem{}
	if sm.Init("test") == true {
		t.Log("create test_spiderHistory success")
	} else {
		t.Fatal("create test_spiderHistory failed")
		return
	}

	key := "xyz"
	if sm.Exist(key) == true {
		t.Fatal("#1 key should not exist!")
	} else {
		t.Log("#1 pass")
	}
	sm.Set(spider.ResourceInfo{Md5: key, ResourceInfo: "test"})
	if sm.Exist(key) == false {
		t.Fatal("#2 key should exist!")
	} else {
		t.Log("#2 pass")
	}

	val := sm.Get(key)

	if !(val.Md5 == key && val.ResourceInfo == "test") {
		t.Fatal("value not right:", val)
	} else {
		t.Log("#3 pass")
	}

	sm.Save()

	sm2 := spider.SpiderHistoryMem{}
	sm2.Init("test")
	if sm2.Exist(key) == false {
		t.Fatal("#4 save failed")
	} else {
		t.Log("#4 pass")
	}
}

func TestResourceString(t *testing.T) {
	r := spider.ResourceInfo{
		Md5:          "123",
		ResourceUrl:  "abc",
		ResourceInfo: "efg",
	}

	if r.String() == "{Md5: 123, ResourceUrl: abc, ResourceInfo: efg, }" {
		t.Log("TestResourceString PASS! output is:", r.String())
	} else {
		t.Fatal("TestResourceString FAILED! output is:", r.String())
	}
}

func TestFindTurnPage(t *testing.T) {
	href := "/sequare/hot/65286/26498?vpageId=2"
	patt := `vpageId=(\d+)`
	if l, r, e := spider.FindTurnPage(href, patt); e == nil {
		expL := len(href) - 1
		expR := expL + 1
		if l != expL || r != expR {
			t.Fatalf("match loc not right: %d != %d, %d != %d", l, expL, r, expR)
		}
		t.Logf("match loc : %d == %d, %d == %d", l, expL, r, expR)
		t.Logf("leftPart: %s, rightPart: %s", href[:l], href[r:])
	} else {
		t.Fatal("TestFindTurnPage FAILED, ERROR:", e)
	}
	href = "/sequare/hot/65286/26498?vpageId=2abc"
	if l, r, e := spider.FindTurnPage(href, patt); e == nil {
		expL := len(href) - 4
		expR := expL + 1
		if l != expL || r != expR {
			t.Fatalf("match loc not right: %d != %d, %d != %d", l, expL, r, expR)
		}
		t.Logf("match loc : %d == %d, %d == %d", l, expL, r, expR)
		t.Logf("leftPart: %s, rightPart: %s", href[:l], href[r:])
	} else {
		t.Fatal("TestFindTurnPage FAILED, ERROR:", e)
	}
}

func TestEncodingHtml(t *testing.T) {
	url := "http://photo.poco.cn/like//index-upi-tpl_type-hot-channel_id-10.html#list/index-upi-p-6-tpl_type-hot-channel_id-10.html#list"
	if doc, err := spider.OpenUtf8Html(url); err == nil {
		infoText := "div ul.item-nav.f-tdn.fl li.cur>a.click"
		tagName := doc.Find(infoText).First().Text()
		if tagName == "美食" {
			t.Log("TestEncodingHtml pass:", tagName)
		} else {
			t.Fatal("TestEncodingHtml pass:", tagName)
		}
	} else {
		t.Fatal("TestEncodingHtml fail with not open url")
	}
}

func TestMd5Sum(t *testing.T) {
	url := "http://image17-c.poco.cn/mypoco/myphoto/20150505/13/1745744532015050513254007_145.jpg?683x1024_120"
	md5sum := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	t.Log("md5sum for ", url, " is ", md5sum)
}
