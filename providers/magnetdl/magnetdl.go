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

	"github.com/stl3/torrodle/models"
	"github.com/stl3/torrodle/request"
)

const (
	Name = "magnetdl"
	Site = "https://www.magnetdl.com"
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
		All: "/s/%v/%d/",
		// Movie: "/search/%v/3000000/%d/seeders",
		// TV:    "/search/%v/2000000/%d/seeders",
		// Porn:  "/search/%v/5000000/%d/seeders",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {

	logrus.Infof("MagnetDL: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("MagnetDL: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	// 	resultsContainer := doc.Find("tbody tr")

	// 	resultsContainer.Each(func(_ int, result *goquery.Selection) {
	// 		title := result.Find("td.text-wrap a").AttrOr("title", "")
	// 		if containsHTMLEncodedEntities(title) {
	// 			decodedTitle, err := decodeHTMLText(title)
	// 			if err != nil {
	// 				logrus.Errorln("Error decoding HTML text:", err)
	// 				// return
	// 				decodedTitle = title
	// 			}
	// 			logrus.Infof("Decoded Title: %s", decodedTitle)
	// 		} else {
	// 			logrus.Infof("Title: %s", title)
	// 		}
	// 		URL, _ := result.Find("td.text-wrap a").Attr("href")
	// 		filesizeStr := result.Find("td:nth-child(3)").Text()
	// 		filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))
	// 		seedersStr := result.Find("td").Eq(4).Text()
	// 		seeders, _ := strconv.Atoi(seedersStr)
	// 		leechersStr := result.Find("td").Eq(5).Text()
	// 		leechers, _ := strconv.Atoi(leechersStr)

	// 		source := models.Source{
	// 			From:     "MagnetDL",
	// 			Title:    title,
	// 			URL:      Site + URL,
	// 			Seeders:  seeders,
	// 			Leechers: leechers,
	// 			FileSize: int64(filesize),
	// 		}
	// 		sources = append(sources, source)
	// 	})

	// 	logrus.Debugf("MagnetDL: [%d] Amount of results: %d", page, len(sources))
	// 	*results = append(*results, sources...)
	// 	wg.Done()
	// }

	// // resultsContainer := doc.Find("#content > div.fill-table > table > tbody tr")

	// // resultsContainer.Each(func(_ int, result *goquery.Selection) {
	// // 	// Parsing title
	// // 	title := result.Find("td.n a").AttrOr("title", "")
	// // 	if containsHTMLEncodedEntities(title) {
	// // 		decodedTitle, err := decodeHTMLText(title)
	// // 		if err != nil {
	// // 			logrus.Errorln("Error decoding HTML text:", err)
	// // 			// return
	// // 			decodedTitle = title
	// // 		}
	// // 		logrus.Infof("Decoded Title: %s", decodedTitle)
	// // 	} else {
	// // 		logrus.Infof("Title: %s", title)
	// // 	}
	// // 	// Parsing URL
	// // 	URL, _ := result.Find("td.n a").Attr("href")

	// // 	// Parsing size
	// // 	filesizeStr := result.Find("td:nth-child(6)").Text()
	// // 	filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))

	// // 	// Parsing seeders
	// // 	seedersStr := result.Find("td.s").Text()
	// // 	seeders, _ := strconv.Atoi(seedersStr)

	// // 	// Parsing leechers
	// // 	leechersStr := result.Find("td.l").Text()
	// // 	leechers, _ := strconv.Atoi(leechersStr)

	// // 	// Parsing magnet link
	// // 	magnet, _ := result.Find("td.m a").Attr("href")

	// // 	source := models.Source{
	// // 		From:     "MagnetDL",
	// // 		Title:    title,
	// // 		URL:      Site + URL,
	// // 		Seeders:  seeders,
	// // 		Leechers: leechers,
	// // 		FileSize: int64(filesize),
	// // 		Magnet:   magnet,
	// // 	}
	// // 	sources = append(sources, source)
	// // })

	// resultsContainer := doc.Find("#content > div.fill-table > table > tbody")
	// resultsContainer := doc.Find("table#download > tbody")
	resultsContainer := doc.Find("#content > div.fill-table > table")

	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		// Extract information from each search result item
		// title := result.Find("tr:nth-child(1) > td.n > a").Text()
		title := result.Find("#content > div.fill-table > table > tbody > tr:nth-child(1) > td.n > a").Text()
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
		URL, _ := result.Find("#content > div.fill-table > table > tbody > tr:nth-child(1) > td.n > a").Attr("href")
		sizeStr := result.Find("#content > div.fill-table > table > tbody > tr:nth-child(1) > td:nth-child(6)").Text()
		size, err := humanize.ParseBytes(sizeStr)
		seeders, _ := strconv.Atoi(result.Find("#content > div.fill-table > table > tbody > tr:nth-child(1) > td.s").Text())
		leechers, _ := strconv.Atoi(result.Find("#content > div.fill-table > table > tbody > tr:nth-child(1) > td.l").Text())
		magnet, _ := result.Find("#content > div.fill-table > table > tbody > tr:nth-child(1) > td.m > a").Attr("href")

		if err != nil {
			fmt.Println("Error converting sizeStr to int:", err)
			fmt.Println("Size string:", sizeStr)
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
