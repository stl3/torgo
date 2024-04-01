package btdigg

import (
	"fmt"
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
	Name = "BTDigg"
	Site = "https://btdig.com"
)

// var Site string // Package-level variable

type provider struct {
	models.Provider
}

func New() models.ProviderInterface {
	// var Site string

	provider := &provider{}
	provider.Name = Name
	provider.Site = Site
	provider.Categories = models.Categories{
		All: "/search?q=%v&order=2&p=%d",
		// All: "/search?q=%v",
		// All: "/search?q=%v&category=all&orderby=seeders&p=%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {

	logrus.Infof("BTDigg: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("BTDigg: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	// Log the HTML content
	// logrus.Debugf("BTDigg: [%d] HTML Content:\n%s", page, html)

	// Find the container of each search result item
	resultsContainer := doc.Find("div.one_result")
	logrus.Debugf("BTDigg: [%d] Number of result containers found: %d", page, resultsContainer.Length())
	// Iterate over each search result item
	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		// Extract information from each search result item
		title := result.Find("div.torrent_name a").Text()
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
		URL, _ := result.Find("div.torrent_name a").Attr("href")
		filesizeStr := result.Find("span.torrent_size").Text()
		filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))
		magnet, _ := result.Find("div.torrent_magnet a").Attr("href")

		source := models.Source{
			From:     "BTDigg",
			Title:    title,
			URL:      Site + URL,
			Seeders:  0, // this site gives no seeder/leecher info
			Leechers: 0,
			FileSize: int64(filesize),
			Magnet:   magnet,
		}
		sources = append(sources, source)

		// logrus.Debugf("BTDigg: [%d] Result %d - Title: %s, URL: %s, FileSize: %d bytes, Magnet: %s",
		// 	page, index+1, source.Title, source.URL, source.FileSize, source.Magnet)
	})

	logrus.Debugf("BTDigg: [%d] Amount of results: %d", page, len(sources))
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
