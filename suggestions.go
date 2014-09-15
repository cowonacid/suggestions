package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"html/template"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
)

func getSeries() map[string]string {

	pattern := regexp.MustCompile("\\[(\\w+)\\]\\s(\\S+)\\.S01E0[1234].*")
	names := map[string]string{}

	doc, err := goquery.NewDocument("http://serienjunkies.org/xml/feeds/episoden.xml")
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("item").Each(func(i int, s *goquery.Selection) {
		title := s.Find("title").Text()

		match := pattern.FindStringSubmatch(title)
		if match == nil {
			return
		}

		lang, series := match[1], strings.Replace(match[2], ".", " ", -1)

		_, exist := names[series]
		if !exist {
			names[series] = lang
		}
	})

	return names
}

type Series struct {
	Name, Language, Description, ImageUrl string
	Ok                                    bool
}

func gatherSeriesInformation(series, lang string, res chan Series) {
	s := Series{Name: series, Language: lang}
	tvdb_url := fmt.Sprintf(
		"http://thetvdb.com/api/GetSeries.php?seriesname=%s&language=de",
		url.QueryEscape(series))

	doc, err := goquery.NewDocument(tvdb_url)
	if err == nil {
		sel := doc.Find("Series").First()
		s.Description = sel.Find("Overview").Text()
		if s.Description != "" {
			s.Ok = true
		}

		banner := sel.Find("banner").Text()
		s.ImageUrl = fmt.Sprintf("http://www.thetvdb.com/banners/%s", banner)
	} else {
		log.Println(err)
	}

	res <- s
}

func series() []Series {
	series_data := getSeries()
	result_chan := make(chan Series)

	for series, lang := range series_data {
		go gatherSeriesInformation(series, lang, result_chan)
	}

	data := []Series{}

	for i := 0; i < len(series_data); i++ {
		s := <-result_chan
		if s.Ok {
			data = append(data, s)
		}
	}

	return data
}

// Get index template from bindata
func buildTemplate() *template.Template {
	html, _ := template.New("index.tmpl").Parse(indexTemplate)
	return html
}

func main() {
	router := gin.Default()
	router.SetHTMLTemplate(buildTemplate())

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.tmpl", series())
	})

	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = ":8080"
	}

	router.Run(listen)
}
