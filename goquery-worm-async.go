package main

import (
	"github.com/PuerkitoBio/goquery"
	"fmt"
	"os"
	"io"
	"strings"
	"container/list"
	"time"
	"regexp"
	"sync"
)

var dirRoot = "D:/study/mavtest"
var siteUrl = "https://m.babytree.com"

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type AgeGroup struct {
	age string
	urls *list.List
}

type Url struct {
	name string
	url string
}

type Content struct {
	contentList []string
}

/**
 * 爬取年龄段
 */
func getAgeGroup(url string) *list.List{
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil
	}

	ageGroups := list.New()
	doc.Find(".cateList").Each(func(i int, s *goquery.Selection) {
		age := s.Find("p").Text()
		urls := list.New()
		s.Find("dd").Each(func(j int, selection *goquery.Selection) {
			selection.Find("a").Each(func(k int, content *goquery.Selection) {
				url := Url{}
				url.name = content.Text()
				link,_ := content.Attr("href")
				url.url = link
				urls.PushBack(url)
			})
		})

		ageGroup := AgeGroup{}
		ageGroup.age = age
		ageGroup.urls = urls
		ageGroups.PushBack(ageGroup)
	})

	fmt.Println(ageGroups)
	return ageGroups
}

/**
 * 爬取文章列表
 */
func getArticleUrls(url string,articleUrls *list.List) *list.List{
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil
	}

	if articleUrls.Len() <=0 {
		articleUrls = list.New()
	}

	doc.Find(".result-box").Each(func(i int, s *goquery.Selection) {
		//文章
		s.Find("a").Each(func(k int, content *goquery.Selection) {
			url := Url{}
			url.name = content.Find(".title").Text()
			link,_ := content.Attr("href")
			url.url = link
			articleUrls.PushBack(url)
		})

		//下一页
		s.Find(".pagination").Each(func(i int, s *goquery.Selection) {
			s.Find("a").Each(func(k int, content *goquery.Selection) {
				if strings.EqualFold("下一页",content.Text()) {
					link,_ := content.Attr("href")
					getArticleUrls(siteUrl+link,articleUrls)
				}
			})
		})
	})
	return articleUrls
}

/**
 * 爬取文章内容
 */
func getContentFromHtml(url string) *Content{
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil
	}

	var contentList []string
	doc.Find(".detail-wrap").Each(func(i int, s *goquery.Selection) {
		name := s.Find("h1").Text()
		contentList = append(contentList, name + "\r\n")
		s.Find(".wrap .content").Each(func(j int, selection *goquery.Selection) {
			contents := selection.Find("p,h2").Next()
			contents.Each(func(k int, content *goquery.Selection) {
				line := content.Text()
				contentList = append(contentList, line + "\r\n")
			})
		})
	})

	if len(contentList) <= 0 {
		doc.Find(".detail-box").Each(func(i int, s *goquery.Selection) {
			name := s.Find("h1").Text()
			contentList = append(contentList, name + "\r\n")

			var part []string
			s.Find("p").Each(func(k int, content *goquery.Selection) {
				line := content.Text()
				if len(strings.TrimSpace(line)) > 0 {
					part = append(part, line + "\r\n")
				}
			})

			if len(part) <= 0 {
				line := s.Text()
				if len(strings.TrimSpace(line)) > 0 {
					part = append(part, line + "\r\n")
				}
			}

			for i := 0; i <len(part) ; i++  {
				contentList = append(contentList, part[i])
			}
		})
	}

	content := Content{}
	content.contentList = contentList
	return &content
}

/**
 * 判断文件是否存在  存在返回 true 不存在返回false
 */
func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func mkdir(path string)  {
	//dir, _ := os.Getwd()  //当前的目录
	if !checkFileIsExist(path){
		err := os.Mkdir(path, 0700)  //在当前目录下生成md目录
		check(err)
	}
}

/**
 * 生成文件
 */
func createFileByContent(dir string,name string,contentList []string)  {
	reg := regexp.MustCompile(`[?*:"<>\/|]`)
	name = reg.ReplaceAllString(name,"-")

	var filename = dir + name +".txt"
	var f *os.File
	var err error

	if checkFileIsExist(filename) { //如果文件存在
		f, err = os.OpenFile(filename, os.O_APPEND, 0666) //打开文件
		err = os.Remove(filename)
	}

	f, err = os.Create(filename) //创建文件
	check(err)
	_, err = io.WriteString(f, strings.Join(contentList,"")) //写入文件(字符串)
	check(err)
	f.Close() //关闭文件
}

func queryAticleAndCreateFiles()  {
	start := time.Now()
	if checkFileIsExist(dirRoot) {
		os.Remove(dirRoot)
	}
	mkdir(dirRoot+"/")
	ageGroups := getAgeGroup(siteUrl+"/learn/?trf=general")

	var wg sync.WaitGroup

	var fileNum = 0
	for t := ageGroups.Front(); t != nil; t = t.Next() {
		ageGroup := t.Value.(AgeGroup)
		dir1 := dirRoot + "/"+ ageGroup.age
		mkdir(dir1)
		for e := ageGroup.urls.Front(); e != nil; e = e.Next() {
			link := e.Value.(Url)
			folder := strings.Replace(strings.TrimSpace(link.name),"/"," ",-1)
			dir2 := dir1 +"/" + folder+"/"
			mkdir(dir2)
			artilceUrl := siteUrl+"/learn/"+link.url
			wg.Add(1)
			go func() {
				defer wg.Done()
				articles := getArticleUrls(artilceUrl,list.New())
				for m := articles.Front(); m != nil; m = m.Next()  {
					article := m.Value.(Url)
					content := getContentFromHtml(siteUrl+article.url)
					if len(strings.TrimSpace(article.name)) > 0 {
						go createFileByContent(dir2,strings.TrimSpace(article.name),content.contentList)
						fileNum++
					}
				}
			}()
		}
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Println("共计耗时：",elapsed)
	fmt.Println("文件个数：",fileNum)
}

func main() {
	queryAticleAndCreateFiles()
}