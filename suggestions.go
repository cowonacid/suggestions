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

func getSeries() (map[string]string, []string) {
	pattern := regexp.MustCompile("\\[(\\w+)\\]\\s(\\S+)\\.S01E0[1234].*")
	names := map[string]string{}
	names_order := []string{}

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
			names_order = append(names_order, series)
		}
	})

	return names, names_order
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
	series_names, series_order := getSeries()
	result_chan := make(chan Series)

	// fetch information for each series concurrently
	for _, series := range series_order {
		go gatherSeriesInformation(series, series_names[series], result_chan)
	}

	// collect series information
	series_data := map[string]Series{}
	for i := 0; i < len(series_names); i++ {
		s := <-result_chan
		series_data[s.Name] = s
	}

	// return complete series information in order where it is found
	data := []Series{}
	for _, series := range series_order {
		s := series_data[series]
		if s.Ok {
			data = append(data, s)
		}
	}
	return data
}

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
