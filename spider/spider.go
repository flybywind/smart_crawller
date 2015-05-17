package spider

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
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
	IsContainer bool
	InfoText    string
	TurnPage    string
	SiblingNum  int
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
	// init spd.waitParseChan by root node
	for _, target := range spd.targetSite {
		spd.doParse(target, spd.waitParseChan)
	}

	go func() {
		for node := range spd.waitParseChan {
			checkDeadLock.Println("entry waitParseChan")
			spd.workersChan <- node
		}
	}()

	for i := 0; i < spd.threadNum; i++ {
		go spd.waitParseNext(spd.waitParseChan, spd.workersChan)
	}
	for i := 0; i < spd.threadNum; i++ {
		<-spd.doneChan
		log.Println("worker finish:", i)
	}

}

func (s Spider) waitParseNext(waitParseChan chan<- SiteNode,
	workersChan <-chan SiteNode) {
	for {
		go func() {
			if err := recover(); err != nil {
				log.Println("Recover in spider parsing, Error:", err)
			}
		}()
		select {
		case curNode := <-workersChan:
			checkDeadLock.Println("entry workersChan")
			s.doParse(curNode, waitParseChan)
		default:
			checkDeadLock.Println("Finish one worker!")
			s.doneChan <- true
			return
		}
	}
}

func (s Spider) doParse(curNode SiteNode, waitParseChan chan<- SiteNode) {
	doc, err := OpenUtf8Html(curNode.Url)
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
				if s.spiderMem.Exist(md5sum) {
					log.Println("skip older one:", url)
					return
				}
				fmt.Println("trying get img:", url)
				client := http.Client{}

				if resp, err := client.Get(url); err == nil {
					body := resp.Body
					defer body.Close()
					savePath := s.storePath + "/" + md5sum + "." + suffix
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
							log.Println("save image:", url, "as", savePath, " failed!\nError:", err)
						}
					} else {
						log.Println("read url["+url+"] with bytes len =", len(buffer), ", error:\n", err)
					}
				} else {
					log.Println("get url:", url, "fail, \n", err)
				}
			}
		})

	} else {
		allParseOut := []SiteNode{}
		if curNode.TurnPage == "" {
			doc.Find(sel).Each(func(i int, sel *goquery.Selection) {
				if href, exist := sel.Attr("href"); exist {
					nextNode := curNode.ChildNode()
					nextNode.Url = GetFullNormalizeUrl(curNode.Url, href)
					allParseOut = append(allParseOut, nextNode)
				}
			})
		} else {
			turnSel := doc.Find(sel).First()
			if href, exist := turnSel.Attr("href"); exist {
				leftIndx, rightIndx, e := FindTurnPage(href, curNode.TurnPage)
				if e == nil {
					// 往后翻页
					for i := 1; i < curNode.SiblingNum+1; i++ {
						nextNode := curNode.ChildNode()
						nextNode.Url = GetFullNormalizeUrl(curNode.Url, fmt.Sprintf("%s%d%s",
							href[:leftIndx], i, href[rightIndx:]))
						allParseOut = append(allParseOut, nextNode)
						checkDeadLock.Println("create new sibling url:", nextNode.Url)
					}
				} else {
					log.Fatal("FindTurnPage not match!")
				}
			} else {
				log.Fatal("cant find href for turn page in ", curNode.Url, " selector: ", sel)
			}
		}

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

func GetFullNormalizeUrl(base string, part string) string {
	partNorm := part // err := purell.NormalizeURLString(part, purell.FlagsUsuallySafeGreedy)
	if len(partNorm) > 8 &&
		(partNorm[0:7] == "http://" ||
			partNorm[0:8] == "https://") {
		return partNorm
	}

	if len(part) > 0 && part[0] == '/' {
		seg := strings.Split(base, "/")
		host := strings.Join(seg[0:3], "/")
		/*ret, err := purell.NormalizeURLString(host+part, purell.FlagsUsuallySafeGreedy)
		if err != nil {
			panic(err)
		}*/
		//return strings.ToLower(ret)
		return host + part
	} else {
		/*ret, err := purell.NormalizeURLString(base+"/"+part, purell.FlagsUsuallySafeGreedy)
		if err != nil {
			panic(err)
		}
		return strings.ToLower(ret)*/
		return base + "/" + part
	}
}

func FindTurnPage(href, regex string) (leftIndx, rightIndx int, err error) {
	patt := regexp.MustCompile(regex)
	matchIndx := patt.FindStringSubmatchIndex(href)

	if matchIndx != nil && len(matchIndx) == 4 {
		leftIndx = matchIndx[2]
		rightIndx = matchIndx[3]
		return leftIndx, rightIndx, nil
	} else {
		return -1, -1, fmt.Errorf("not found turn page")
	}

}

func OpenUtf8Html(url string) (*goquery.Document, error) {
	client := http.Client{}
	if resp, err := client.Get(url); err == nil {
		body := resp.Body
		if rawBytes, err := ioutil.ReadAll(body); err != nil {
			return nil, err
		} else {
			newReader := bytes.NewReader(rawBytes)
			body.Close()
			utf8Reader, err := charset.NewReader(newReader, "gbk")
			if err == nil {
				return goquery.NewDocumentFromReader(utf8Reader)
			} else {
				return nil, err
			}
		}
	} else {
		return nil, err
	}
}
