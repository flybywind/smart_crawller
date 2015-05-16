package spider

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var checkDeadLock = log.New(os.Stderr, "CheckDeadLock:",
	log.LstdFlags|log.Lshortfile)

// 用户定义的网站结构
type SiteNode struct {
	Url           string
	ChildSelector string
	ChildNode     func() SiteNode
	// 如果是container，那么NextSelector就是target了
	// InfoText就可能有值
	IsContainer   bool
	ContainerId int
	InfoText      string
	TurnPage string
}

func (n *SiteNode) String() string {
	V := reflect.ValueOf(n).Elem()
	T := reflect.TypeOf(n).Elem()
	repr := bytes.NewBufferString("{")
	for i := 0; i < T.NumField(); i++ {
		repr.WriteString(fmt.Sprintf("%s: %v, ", T.Field(i).Name, V.Field(i).Interface()))
	}
	repr.WriteString("}")
	return repr.String()
}

// 抓取历史
type SpiderHistory interface {
	Init()
	Set(url string)
	Query(url string) bool
}

// 用gobson保存历史信息
type SpiderHistoryMem struct {
	session   *mgo.Session
	collector *mgo.Collection
	mem       map[string]ResourceInfo
}

func (h *SpiderHistoryMem) Init(c string) bool {
	session, err := mgo.Dial("spider:123456@127.0.0.1/spider")

	if err == nil {
		h.session = session
		h.collector = h.session.DB("spider").C(c)
		return true
	} else {
		log.Fatal("Create SpiderHistoryMem failed, Error:", err)
		return false
	}
}

func (h *SpiderHistoryMem) Exist(k string) bool {
	selector := bson.M{"md5": k}
	ret := ResourceInfo{}
	e := h.collector.Find(selector).One(&ret)
	return e == nil
}

func (h *SpiderHistoryMem) Get(k string) ResourceInfo {
	selector := bson.M{"md5": k}
	ret := ResourceInfo{}
	if e := h.collector.Find(selector).One(&ret); e == nil {
		return ret
	} else {
		return ResourceInfo{}
	}
}
func (h *SpiderHistoryMem) Set(v ResourceInfo) {
	h.collector.Insert(v)
}

func (h SpiderHistoryMem) Save() {
	h.session.Close()
}

/////////////////////

// 每个资源的信息，包括保持路径，说明信息，类型
type ResourceInfo struct {
	Md5          string
	ResourceUrl  string
	ResourceInfo string
}

func (r *ResourceInfo) String() string {
	V := reflect.ValueOf(r).Elem()
	T := reflect.TypeOf(r).Elem()
	repr := bytes.NewBufferString("{")
	for i := 0; i < T.NumField(); i++ {
		repr.WriteString(fmt.Sprintf("%s: %v, ", T.Field(i).Name, V.Field(i).Interface()))
	}
	repr.WriteString("}")
	return repr.String()
}

type Spider struct {
	targetSite    []SiteNode
	threadNum     int
	storePath     string
	spiderMem     SpiderHistoryMem
	waitParseChan chan SiteNode
	workersChan   chan SiteNode
	doneChan      chan bool
}

func NewSpider(targets []SiteNode, thread_num int, store_path string) *Spider {
	spd := new(Spider)
	spd.threadNum = thread_num
	spd.storePath = store_path
	spd.targetSite = targets

	spd.workersChan = make(chan SiteNode, spd.threadNum)
	spd.waitParseChan = make(chan SiteNode, spd.threadNum*20)
	spd.doneChan = make(chan bool, spd.threadNum)
	return spd
}
func (spd Spider) Run() {
	spd.spiderMem.Init("spider_mem")
	defer spd.spiderMem.Save()
	go func() {
		for _, target := range spd.targetSite {
			spd.waitParseChan <- target
		}
	}()

	go func() {
		for node := range spd.waitParseChan {
			checkDeadLock.Println("entry waitParseChan")
			spd.workersChan <- node
		}
	}()

	for i := 0; i < spd.threadNum; i++ {
		go spd.parseNext(spd.waitParseChan, spd.workersChan)
	}
	for i := 0; i < spd.threadNum; i++ {
		<-spd.doneChan
		log.Println("worker finish:", i)
	}

}

func (s Spider) parseNext(waitParseChan chan<- SiteNode,
	workersChan <-chan SiteNode) {
	for curNode := range workersChan {
		checkDeadLock.Println("entry workersChan")
		go func() {
			if err := recover(); err != nil {
				log.Fatal("Recover in spider parsing, Error:", err)
			}
		}()
		doc, err := goquery.NewDocument(curNode.Url)
		if err != nil {
			// handle error
			log.Println("get url[", curNode.Url, "] failed!")
			return
		}
		checkDeadLock.Println("Open url:", curNode.Url)
		sel := curNode.ChildSelector
		if curNode.IsContainer {
			seg := strings.Split(sel, "[")
			if len(seg) < 2 {
				panic("容器中的目标资源选择子格式错误，期望格式为： selector[attr][suffix]!")
			}
			sel := seg[0]
			attr := seg[1][0 : len(seg[1])-1]
			defaultSuffix := ""
			if len(seg) == 3 {
				defaultSuffix = seg[2][0 : len(seg[2])-1]
			}
			tagInfo := doc.Find(curNode.InfoText).Text()
			tagInfo = strings.TrimSpace(tagInfo)
			doc.Find(sel).Each(func(i int, sel *goquery.Selection) {
				if attrVal, exist := sel.Attr(attr); exist {
					url := GetFullNormalizeUrl(curNode.Url, attrVal)
					md5sum := fmt.Sprintf("%x", md5.Sum([]byte(url)))
					seg := strings.Split(url, ".")
					suffix := defaultSuffix
					if len(seg) > 1 {
						suffix = seg[len(seg)-1]
					}
					savePath := s.storePath + "/" + md5sum + "." + suffix

					if s.spiderMem.Exist(md5sum) {
						log.Println("skip older one:", url)
						return
					}
					fmt.Println("trying get img:", url)
					client := http.Client{}
					if resp, err := client.Get(url); err == nil {
						body := resp.Body
						defer body.Close()
						if buffer, err := ioutil.ReadAll(body); len(buffer) > 0 && err == nil {
							if err := ioutil.WriteFile(savePath, buffer, 0666); err == nil {
								info := ResourceInfo{
									Md5:          md5sum,
									ResourceUrl:  url,
									ResourceInfo: tagInfo,
								}
								s.spiderMem.Set(info)
								log.Println("save image:", url, "as", savePath, "success! info is:\n"+info.String())
							} else {
								log.Fatal("save image:", url, "as", savePath, " failed!\nError:", err)
							}
						} else {
							log.Fatal("read url["+url+"] with bytes len =", len(buffer), ", error:\n", err)
						}
					} else {
						log.Fatal("get url:", url, "fail, \n", err)
					}
				}
			})

			if curNode.TurnPage != "" {
					nextPageUrl := fmt.Sprintf("%s%s%d", curNode.Url, curNode.TurnPage,  curNode.ContainerId+1)
						nextSibNode := new(SiteNode)
						*nextSibNode = curNode
						nextSibNode.Url = nextPageUrl
						go func() {
							waitParseChan <- (*nextSibNode)
							checkDeadLock.Println("send one to waitParseChan:", nextSibNode)
						}()
					}
				}
			}
		} else {
			allParseOut := []SiteNode{}
			doc.Find(sel).Each(func(i int, sel *goquery.Selection) {
				if href, exist := sel.Attr("href"); exist {
					nextNode := curNode.ChildNode()
					nextNode.Url = GetFullNormalizeUrl(curNode.Url, href)
					allParseOut = append(allParseOut, nextNode)
				}
			})
			// parse next out:
			go func() {
				for _, nextNode := range allParseOut {
					waitParseChan <- nextNode
					log.Println("find an intemediat page:", nextNode.Url)
					checkDeadLock.Println("send one to waitParseChan:", nextNode.Url)
				}
			}()
		}
	}
	checkDeadLock.Println("Finish one worker!")
	s.doneChan <- true
}

func GetFullNormalizeUrl(base string, part string) string {
	partNorm, err := purell.NormalizeURLString(part, purell.FlagsUsuallySafeGreedy)
	if err == nil && len(partNorm) > 8 &&
		(partNorm[0:7] == "http://" ||
			partNorm[0:8] == "https://") {
		return strings.ToLower(partNorm)
	}

	if len(part) > 0 && part[0] == '/' {
		seg := strings.Split(base, "/")
		host := strings.Join(seg[0:3], "/")
		ret, err := purell.NormalizeURLString(host+part, purell.FlagsUsuallySafeGreedy)
		if err != nil {
			panic(err)
		}
		return strings.ToLower(ret)
	} else {
		ret, err := purell.NormalizeURLString(base+"/"+part, purell.FlagsUsuallySafeGreedy)
		if err != nil {
			panic(err)
		}
		return strings.ToLower(ret)
	}
}
