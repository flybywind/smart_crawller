package spider

import (
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// 用户定义的网站结构
type SiteNode struct {
	Url          string
	NextSelector string
	NextNode     func() SiteNode
	// 如果是container，那么NextSelector就是target了
	// InfoText就可能有值
	IsContainer bool
	InfoText    string
}

// 抓取历史
type SpiderHistory interface {
	Init()
	Set(url string)
	Query(url string) bool
}

// 用gobson保存历史信息
type SpiderHistoryMem struct {
	path string
	mem  map[string]ResourceInfo
}

func (h *SpiderHistoryMem) Init(p string) {
	h.path = p
	if file, err := os.Open(p); err == nil {
		decoder := gob.NewDecoder(file)
		defer file.Close()
		decoder.Decode(&h.mem)
	} else {
		h.mem = make(map[string]ResourceInfo)
		fmt.Println("SpiderHistoryMem:", p, "not exist, create new !")
	}
}

func (h *SpiderHistoryMem) Exist(k string) bool {
	_, exist := h.mem[k]
	return exist
}

func (h *SpiderHistoryMem) Get(k string) ResourceInfo {
	if h.Exist(k) {
		return h.mem[k]
	} else {
		return ResourceInfo{}
	}
}
func (h *SpiderHistoryMem) Set(k string, v ResourceInfo) {
	v.Md5 = k
	h.mem[k] = v
}

func (h SpiderHistoryMem) Save() bool {
	file, err := os.Create(h.path)
	if err != nil {
		return false
	}
	defer file.Close()
	writer := gob.NewEncoder(file)
	if err := writer.Encode(h.mem); err != nil {
		return false
	} else {
		return true
	}
}

/////////////////////

// 每个资源的信息，包括保持路径，说明信息，类型
type ResourceInfo struct {
	Md5          string
	ResourceUrl  string
	ResourceInfo string
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
	spd.spiderMem.Init("spider.mem")
	defer spd.spiderMem.Save()
	go func() {
		for _, target := range spd.targetSite {
			spd.waitParseChan <- target
		}
	}()

	go func() {
		for node := range spd.waitParseChan {
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
		doc, err := goquery.NewDocument(curNode.Url)
		if err != nil {
			// handle error
			log.Println("get url[", curNode.Url, "] failed!")
			return
		}
		sel := curNode.NextSelector
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
			tagInfo, _ := doc.Has(curNode.InfoText).First().Html()
			fmt.Println("url:", curNode.Url, ", selector:", curNode.InfoText, "TagInfo:\n", tagInfo)
			doc.Add(sel).Each(func(i int, sel *goquery.Selection) {
				if attrVal, exist := sel.Attr(attr); exist {
					url := GetFullNormalizeUrl(curNode.Url, attrVal)
					md5sum := fmt.Sprintf("%x", md5.Sum([]byte(url)))
					seg := strings.Split(url, ".")
					suffix := defaultSuffix
					if len(seg) > 1 {
						suffix = seg[len(seg)-1]
					}
					savePath := s.storePath + "/" + md5sum + "." + suffix
					if s.spiderMem.Exist(savePath) {
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
									ResourceUrl:  url,
									ResourceInfo: tagInfo,
								}
								s.spiderMem.Set(md5sum, info)
								log.Println("save image:", url, "as", savePath, "success! info is:", info)
							} else {
								log.Fatal("save image:", url, "as", savePath, "failed!\n", err)
							}
						} else {
							log.Fatal("read url["+url+"] with bytes len =", len(buffer), ", error:\n", err)
						}
					} else {
						log.Fatal("get url:", url, "fail, \n", err)
					}
				}
			})
		} else {
			allParseOut := []SiteNode{}
			doc.Add(sel).Each(func(i int, sel *goquery.Selection) {
				if href, exist := sel.Attr("href"); exist {
					nextNode := curNode.NextNode()
					nextNode.Url = GetFullNormalizeUrl(curNode.Url, href)
					log.Println("find an intemediat page:", nextNode.Url)
					allParseOut = append(allParseOut, nextNode)
				}
			})
			// parse next out:
			go func() {
				for _, nextNode := range allParseOut {
					waitParseChan <- nextNode
				}
			}()
		}
	}
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
