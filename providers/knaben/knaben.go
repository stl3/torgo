package knaben

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"

	"github.com/stl3/torgo/models"
	"github.com/stl3/torgo/request"
)

const (
	Name = "knaben"
	Site = "https://knaben.eu"
)

type provider struct {
	models.Provider
}

func New() models.ProviderInterface {
	// var Site string

	provider := &provider{}
	provider.Name = Name
	provider.Site = Site
	provider.Categories = models.Categories{
		All:   "/search/%v/0/%d/seeders",
		Movie: "/search/%v/3000000/%d/seeders",
		TV:    "/search/%v/2000000/%d/seeders",
		Porn:  "/search/%v/5000000/%d/seeders",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {

	logrus.Infof("knaben: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("knaben: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("tbody > tr")
	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		title := result.Find("td.text-wrap a").Text()
		if containsHTMLEncodedEntities(title) {
			decodedTitle, err := decodeHTMLText(title)
			if err != nil {
				logrus.Errorln("Error decoding HTML text:", err)
				// return
				decodedTitle = title
			}
			logrus.Infof("Decoded Title: %s", decodedTitle)
		} else {
			logrus.Infof("Title: %s", title)
		}
		
		filesizeStr := result.Find("td:nth-child(3)").Text()
		filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))
		// date := result.Find("td[title^='20']").Text()
		seeders, _ := strconv.Atoi(result.Find("td:nth-last-child(3)").Text())
		leechers, _ := strconv.Atoi(result.Find("td:nth-last-child(2)").Text())
		magnet, _ := result.Find("td.text-wrap a").Attr("href")

		// if err != nil {
		// 	// log.Println("Error converting sizeStr to int:", err)
		// 	log.Println("Size string:", filesizeStr)
		// }

		source := models.Source{
			From:     "knaben",
			Title:    title,
			URL:      surl,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
			// You may add other fields like category, subcategory, date as needed
		}
		sources = append(sources, source)
	})

	logrus.Debugf("knaben: [%d] Amount of results: %d", page, len(sources))
	*results = append(*results, sources...)
	wg.Done()
}

// Checks if the text contains HTML-encoded entities
func containsHTMLEncodedEntities(text string) bool {
	return strings.ContainsAny(text, "&<>'\"")
}

// Decodes HTML-encoded text
func decodeHTMLText(text string) (string, error) {
	var decodedText string
	tokenizer := html.NewTokenizer(strings.NewReader(text))

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err != nil {
				return text, err // Return the original text and the decoding error
			}
			return decodedText, nil // Return the decoded text
		case html.TextToken:
			token := tokenizer.Token()
			decodedText += token.Data
		}
	}
}
