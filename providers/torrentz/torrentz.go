package torrentz

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
	Name = "Torrentz2"
	// Site = "https://torrentz2.eu"
	Site = "https://torrentz2.nz"
)

type provider struct {
	models.Provider
}

func New() models.ProviderInterface {
	provider := &provider{}
	provider.Name = Name
	provider.Site = Site
	provider.Categories = models.Categories{
		All:   "/search?q=%v&page=%d",
		Movie: "/search?q=%v&page=%d",
		TV:    "/search?q=%v&page=%d",
		Anime: "/search?q=%v&page=%d",
		Porn:  "/search?q=%v&page=%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 0, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	logrus.Infof("Torrentz2: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("Torrentz2: [%d]", page), err)
		wg.Done()
		return
	}
	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	resultsContainer := doc.Find("div.results dl")

	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		// Extract information from each search result item
		title := result.Find("dt a").Text()
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
		URL, _ := result.Find("dt a").Attr("href")

		filesizeStr := result.Find("dd span:nth-child(3)").Text()
		filesize, _ := humanize.ParseBytes(filesizeStr)

		seedersStr := result.Find("dd span:nth-child(4)").Text()
		seeders, _ := strconv.Atoi(seedersStr)

		leechersStr := result.Find("dd span:nth-child(5)").Text()
		leechers, _ := strconv.Atoi(leechersStr)

		magnet, _ := result.Find("dd a i.fa-magnet").Parent().Attr("href")

		source := models.Source{
			From:  "Torrentz2",
			Title: title,
			// URL:      Site + URL,
			URL:      URL,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("Torrentz2: [%d] Amount of results: %d", page, len(sources))
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
			// return decodedText, nil // Return the decoded text
			return html.UnescapeString(decodedText), nil // Use UnescapeString on the decoded text
		case html.TextToken:
			token := tokenizer.Token()
			decodedText += token.Data
		}
	}
}
