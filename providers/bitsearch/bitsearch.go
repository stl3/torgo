package bitsearch

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
	Name = "Bitsearch"
	Site = "https://bitsearch.to"
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
		// All:   "/search/all/%v/seeds/%d",
		All:   "/search?q=%v&page=%d",
		Movie: "/search?q=%v&page=%d&category=1",
		TV:    "/search?q=%v&page=%d",
		Anime: "/search?q=%v&page=%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	// func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup, Site string) { // Add Site as a parameter

	logrus.Infof("Bitsearch: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("Bitsearch: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("li.card.search-result")

	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		// Extract information from each search result item
		title := result.Find("h5.title a").Text()
		// Output the decoded title if needed
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

		URL, _ := result.Find("h5.title a").Attr("href")
		// Extract other relevant information
		filesizeStr := result.Find("div.stats div:nth-child(2)").Text()
		filesize, err := humanize.ParseBytes(filesizeStr)
		seedersStr := result.Find("div.stats div:nth-child(3) font").Text()
		seeders, _ := strconv.Atoi(seedersStr)
		leechersStr := result.Find("div.stats div:nth-child(4) font").Text()
		leechers, _ := strconv.Atoi(leechersStr)

		if err != nil {
			// Print the leechers string for debugging
			fmt.Println("Error converting leechersStr to int:", err)
			fmt.Println("Leechers string:", leechersStr)
		}
		// fmt.Println("Leechers count:", leechers)
		magnet, _ := result.Find("div.links a.dl-magnet").Attr("href")

		source := models.Source{
			From:     "Bitsearch",
			Title:    title,
			URL:      Site + URL,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("Bitsearch: [%d] Amount of results: %d", page, len(sources))
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
