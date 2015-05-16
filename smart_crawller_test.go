package smart_crawller_test

import (
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
