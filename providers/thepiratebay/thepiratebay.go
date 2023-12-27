package thepiratebay

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"

	"github.com/stl3/torrodle/models"
	"github.com/stl3/torrodle/request"
)

const (
	Name = "ThePirateBay"
	// Site = "https://thepiratebay.org"
	Site = "https://prbay.top"
	// Site = "https://thepiratebay7.com/"
)

type provider struct {
	models.Provider
}

func New() models.ProviderInterface {
	provider := &provider{}
	provider.Name = Name
	provider.Site = Site
	provider.Categories = models.Categories{
		All:   "/search/%v/%d/99/0",
		Movie: "/search/%v/%d/99/201",
		TV:    "/search/%v/%d/99/205",
		Porn:  "/search/%v/%d/99/501",
	}
	return provider
}

// func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
// 	results, err := provider.Query(query, categoryURL, count, 30, 0, extractor)
// 	return results, err
// }

// pbay doesn't seem to like queries with spaces, but will gladly accept them
// in places of the spaces - they will still return results that have spaces
// or dots.
func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	// Replace spaces with dots
	encodedQuery := strings.ReplaceAll(query, " ", ".")
	// encodedQuery := url.QueryEscape(query)
	results, err := provider.Query(encodedQuery, categoryURL, count, 30, 0, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	logrus.Infof("ThePirateBay: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("ThePirateBay: [%d]", page), err)
		wg.Done()
		return
	}
	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	table := doc.Find("table#searchResult").Find("tbody")
	table.Find("tr").Each(func(i int, tr *goquery.Selection) {
		tds := tr.Find("td")
		a := tds.Eq(1).Find("a.detLink")
		// title
		title := a.Text()
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
		// seeders
		s := tds.Eq(2).Text()
		seeders, _ := strconv.Atoi(strings.TrimSpace(s))
		// leechers
		l := tds.Eq(3).Text()
		leechers, _ := strconv.Atoi(strings.TrimSpace(l))
		// filesize
		re := regexp.MustCompile(`Size\s(.*?),`)
		text := tds.Eq(1).Find("font").Text()
		matches := re.FindStringSubmatch(text)

		var fs string
		if len(matches) > 1 {
			fs = matches[1]
		}

		filesize, _ := humanize.ParseBytes(strings.TrimSpace(fs)) // convert human words to bytes number
		// url
		URL, _ := a.Attr("href")
		// magnet
		magnet, _ := tds.Eq(1).Find(`a[title="Download this torrent using magnet"]`).Attr("href")
		// ---
		source := models.Source{
			From:  "ThePirateBay",
			Title: strings.TrimSpace(title),
			// URL:      Site + URL,
			URL:      URL,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("ThePirateBay: [%d] Amount of results: %d", page, len(sources))
	if len(sources) > 0 {
		*results = append(*results, sources...)
	}

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
