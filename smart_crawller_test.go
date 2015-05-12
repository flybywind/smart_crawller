package smart_crawller_test

import (
	"smart_crawller/spider"
	"testing"
)

func TestSpiderHistoryMem(t *testing.T) {
	sm := spider.SpiderHistoryMem{}
	sm.Init("test_spiderHistory.mem")

	key := "xyz"
	if sm.Exist(key) == true {
		t.Fatal("#1 key should not exist!")
	} else {
		t.Log("#1 pass")
	}
	sm.Set(key, spider.ResourceInfo{ResourceInfo: "test"})
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
	sm2.Init("test_spiderHistory.mem")
	if sm2.Exist(key) == false {
		t.Fatal("#4 save failed")
	} else {
		t.Log("#4 pass")
	}
}
