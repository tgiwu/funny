package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// var workList = make(map[string] bool)
var tokensC = make(chan struct{}, 5)

// var imageChan = make(chan *image, 5)
var dir = "./manga"

type chapter struct {
	title        string
	dataRedirect string
}

type page struct {
	name string
	link string
}

//type image struct {
//	chapter *chapter
//	page *page
//	name string
//	link string
//}

func main() {
	var workList = make(chan []chapter)

	var n int
	n++

	//chapters := getChapters("https://xxxx.info/manga/manga/manga-1-0?manga-paged=1")

	go func() {
		workList <- readChapterFromFile()
	}()

	for ; n > 0; n-- {
		list := <-workList
		for _, task := range list {
			n++
			go func(task chapter) {
				scanChapter(task)
			}(task)
		}
	}
}

func getChapters(url string) []*chapter {
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx1, cancel1 := context.WithTimeout(ctx, 30*time.Second)
	defer cancel1()

	//var nodes []*cdp.Node
	var selects []*cdp.Node
	var title string

	fmt.Printf("start fetch %s\n", url)
	err := chromedp.Run(ctx1,
		chromedp.Navigate(`https://mangazuki.info/manga/brawling-go/brawling-go-145-0?manga-paged=1`),
		//chromedp.WaitVisible("div .page-break"),
		chromedp.Nodes(`body > div.wrap > div.body-wrap > div > div > div > div > div > div > div > div > div.entry-header > div.wp-manga-nav > div.select-view > div.chapter-selection > div > label > select`, &selects, chromedp.ByQuery),
		//chromedp.Nodes(`div .read-container`, &nodes, chromedp.ByQuery),
		chromedp.Text(`body > div.wrap > div.body-wrap > div > div > div > div > div > div > div > div > div.entry-header > div.wp-manga-nav > div.entry-header_wrap > div > div.c-breadcrumb > ol > li.active`, &title, chromedp.ByQuery),
	)
	if nil != err {
		log.Fatal(err)
	}

	fmt.Printf("title %s \n", title)
	var chapters []*chapter
	if nil != selects {
		chapters = make([]*chapter, len(selects[0].Children))
		fmt.Printf("selects length %d\n", len(selects))
		options := selects[0].Children
		for i := 0; i < len(options); i++ {
			if options[i].NodeName == "OPTION" {
				fmt.Printf("title %s \n", options[i].AttributeValue("value"))
				fmt.Printf("data-redirect %s \n", options[i].AttributeValue("data-redirect"))
				chapters[i] = &chapter{options[i].AttributeValue("value"), options[i].AttributeValue("data-redirect")}
			}
		}
	}
	return chapters
}

func readChapterFromFile() []chapter {
	bytes, err := ioutil.ReadFile("./log/chapter.log")
	if nil != err {
		log.Fatalf("read chapter file %s", err)
	}

	all := string(bytes)
	urls := strings.Split(all, "\n")

	chapters := make([]chapter, len(urls))
	for i, s := range urls {
		seg := strings.Split(s, "/")
		title := strings.TrimSpace(seg[len(seg)-1])
		link := s
		if len(strings.TrimSpace(s)) != 0 {
			fmt.Printf("title %s ::: %s \n", title, link)
			chapters[i] = chapter{title, link}
		}

	}
	return chapters
}

func scanChapter(task chapter) {
	tokensC <- struct{}{}
	folder := strings.Join([]string{dir, task.title}, "/")
	_, err := os.Stat(folder)
	if os.IsNotExist(err) {
		e := os.MkdirAll(folder, 0777)
		if nil != e {
			fmt.Printf("mk %s fatal %s", folder, e)
			<-tokensC
			return
		}
	}

	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	defer cancel()

	var pagesSelects []*cdp.Node
	fmt.Printf("navigate to %s \n", task.dataRedirect)
	err1 := chromedp.Run(ctx, chromedp.Navigate(task.dataRedirect),
		chromedp.Nodes(`#single-pager`, &pagesSelects, chromedp.ByQuery),
	)

	if nil != err1 {
		log.Fatal(err1)
	}

	var pages []*page
	if nil != pagesSelects {
		pages = make([]*page, len(pagesSelects[0].Children))
		fmt.Printf("selects length %d\n", len(pagesSelects))
		options := pagesSelects[0].Children
		for i := 0; i < len(options); i++ {
			if options[i].NodeName == "OPTION" {
				fmt.Printf("title %s \n", options[i].AttributeValue("value"))
				fmt.Printf("data-redirect %s \n", options[i].AttributeValue("data-redirect"))
				pages[i] = &page{options[i].AttributeValue("value"), options[i].AttributeValue("data-redirect")}
			}
		}
	}

	if len(pages) > 0 {
		for i := 0; i < len(pages); i++ {
			var images []*cdp.Node
			err2 := chromedp.Run(ctx,
				chromedp.Navigate(pages[i].link),
				chromedp.Nodes(`img[id^='image-']`, &images, chromedp.ByQueryAll),
			)
			if nil != err2 {
				log.Fatal(err1)
			}
			fmt.Printf("images size %d \n", len(images))
			for j := 0; j < len(images); j++ {

				exName := "jpg"
				fmt.Println(images[j].AttributeValue("src"))
				parts := strings.Split(images[j].AttributeValue("src"), ".")
				if len(parts) >= 2 {
					exName = parts[len(parts)-1]
				}
				logUrl(images[j].AttributeValue("src"))
				fmt.Printf("try to download %s \n", images[j].AttributeValue("src"))
				err3 := downloadFile(images[j].AttributeValue("src"),
					folder+"/"+images[j].AttributeValue("id")+"."+exName,
					func(length, downLen int64) {
						fmt.Printf("download %d / %d \n", downLen, length)
					})

				if err3 != nil {
					fmt.Printf("download fatal %s \n", err3)
				}
			}
		}
	}
	<-tokensC
}

func IsFileExist(filename string, filesize int64) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		fmt.Println(info)
		return false
	}
	if filesize == info.Size() {
		fmt.Println("文件已存在！", info.Name(), info.Size(), info.ModTime())
		return true

	}
	del := os.Remove(filename)
	if del != nil {
		fmt.Println(del)

	}
	return false
}

func logUrl(url string) {
	_, err1 := os.Stat(dir + "/url.log")

	var file *os.File
	if !os.IsExist(err1) {
		_, err2 := os.Create(dir + "/url.log")
		if err2 != nil {
			log.Fatalf("create log fatal %s", err2)
		}
	}
	file, err21 := os.OpenFile(dir+"/url.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err21 != nil {
		log.Fatalf("create log fatal %s", err21)
	}
	defer file.Close()
	i, err := file.Write([]byte(url + "\n"))
	if nil != err {
		log.Fatalf("write log fatal %s\n", err21)
	}

	fmt.Printf("log %d \n", i)
}

func downloadFile(url string, localPath string, fb func(length, downLen int64)) error {

	var (
		fsize   int64
		buf     = make([]byte, 32*1024)
		written int64
	)
	tmpFilePath := localPath + ".download"
	fmt.Println(tmpFilePath)
	//创建一个http client
	client := new(http.Client)
	//client.Timeout = time.Second * 60 //设置超时时间
	//get方法获取资源
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	//读取服务器返回的文件大小
	fsize, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		fmt.Println(err)

	}
	if IsFileExist(localPath, fsize) {
		return err

	}
	fmt.Println("fsize", fsize)
	//创建文件
	file, err := os.Create(tmpFilePath)
	if err != nil {
		return err

	}
	defer file.Close()
	if resp.Body == nil {
		return errors.New("body is null")

	}
	defer resp.Body.Close()
	//下面是 io.copyBuffer() 的简化版本
	for {
		//读取bytes
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			//写入bytes
			nw, ew := file.Write(buf[0:nr])
			//数据长度大于0
			if nw > 0 {
				written += int64(nw)

			}
			//写入出错
			if ew != nil {
				err = ew
				break

			}
			//读取是数据长度不等于写入的数据长度
			if nr != nw {
				err = io.ErrShortWrite
				break

			}

		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
		//没有错误了快使用 callback
		fb(fsize, written)
	}
	fmt.Println(err)
	if err == nil {
		file.Close()
		err = os.Rename(tmpFilePath, localPath)
		fmt.Println(err)
	}
	return err
}
