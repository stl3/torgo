package magnetdl

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
	Name = "magnetdl"
	Site = "https://www.magnetdl.com"
	// Site = "https://magnetdl.skin"
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
		// Changed Jan 2024
		// Format now takes the first letter from query, and changes space to "-"
		// Eg https://www.magnetdl.com/t/the-walking-dead-s11e01/1/
		// https://www.magnetdl.com/first_letter_of_query/%v/%d/
		All:   "/%v/%d/",
		Movie: "/%v/%d/",
		TV:    "/%v/%d/",
		Porn:  "/%v/%d/",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	modifiedQuery := modifyQuery(query)
	logrus.Infof("Modified query before changes: %s", modifiedQuery)
	modifiedQuery = string(query[0]) + "/" + modifiedQuery
	logrus.Infof("Modified query afer changes: %s", modifiedQuery)
	results, err := provider.Query(modifiedQuery, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	// Replace "%2F" with "/"
	surl = strings.ReplaceAll(surl, "%2F", "/")
	// Log or display the full URL before making the request
	logrus.Infof("MagnetDL: [%d] Requesting URL: %s\n", page, surl)

	logrus.Infof("MagnetDL: [%d] Extracting results...\n", page)
	// _, html, err := request.Get(nil, strings.ReplaceAll(surl, "/", "%2F"), nil)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("MagnetDL: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("#content > div.fill-table > table > tbody > tr")
	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		// Extract information from each search result item
		title := result.Find("td.n a").Text()

		if containsHTMLEncodedEntities(title) {
			decodedTitle, err := decodeHTMLText(title)
			if err != nil {
				// logrus.Errorln("Error decoding HTML text:", err)
				// return
				decodedTitle = title
			}
			logrus.Infof("Decoded Title: %s", decodedTitle)
		} else {
			logrus.Infof("Title: %s", title)
		}

		URL, _ := result.Find("td.n a").Attr("href")
		sizeStr := result.Find("td:nth-child(6)").Text()
		size, err := humanize.ParseBytes(sizeStr)
		seeders, _ := strconv.Atoi(result.Find("td.s").Text())
		leechers, _ := strconv.Atoi(result.Find("td.l").Text())
		magnet, _ := result.Find("td.m a").Attr("href")

		if err != nil {
			// log.Println("Error converting sizeStr to int:", err)
			// log.Println("Size string:", sizeStr)
			// logrus.Errorln("Error converting sizeStr to int:", err)
			logrus.Infof("Size string: %s", sizeStr)
		}

		source := models.Source{
			From:     "MagnetDL",
			Title:    title,
			URL:      Site + URL, // Assuming 'Site' is declared somewhere
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(size),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("MagnetDL: [%d] Amount of results: %d", page, len(sources))
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

func modifyQuery(query string) string {
	// Take the first letter of the query
	// firstLetter := string(query[0])

	// Replace spaces with "-"
	modifiedQuery := strings.ReplaceAll(query, " ", "-")
	// finalQuery := firstLetter + "/" + modifiedQuery
	// Construct the modified query

	return modifiedQuery
}
