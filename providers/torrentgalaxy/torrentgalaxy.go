package torrentgalaxy

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
	Name = "torrentgalaxy"
	Site = "https://torrentgalaxy.to"
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
		All:           "/torrents.php?search=%v#results%d",
		Movie:         "/torrents.php?c42=1&search=%v#results%d",
		TV:            "/torrents.php?c41=1&search=%v#results%d",
		Anime:         "/torrents.php?c28=1&search=%v#results%d",
		Documentaries: "/torrents.php?genres[]=5&search=%v#results%d",
		Porn:          "/torrents.php?c35=1&search=%v#results%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	modifiedQuery := modifyQuery(query)
	// logrus.Infof("Query: %s", modifiedQuery)
	results, err := provider.Query(modifiedQuery, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	// Log or display the full URL before making the request
	surl = regexp.MustCompile(`\d+$`).ReplaceAllString(surl, "")

	logrus.Infof("TorrentGalaxy: [%d] Requesting URL: %s\n", page, surl)
	logrus.Infof("TorrentGalaxy: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("TorrentGalaxy: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("div.tgxtablerow.txlight")
	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		title := result.Find("div:nth-child(4) > div > a.txlight > span > b").Text()
		// logrus.Infof("Title: %s", title)
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

		URL, _ := result.Find("div:nth-child(2) > div > a.txlight").Attr("href")
		sizeStr := result.Find("div:nth-child(8) > span").Text()
		size, err := humanize.ParseBytes(sizeStr)
		if err != nil {
			logrus.Infof("Size string: %s", sizeStr)
		}

		seeders, _ := strconv.Atoi(result.Find("div:nth-child(11) > span > font:nth-child(1) > b").Text())
		leechers, _ := strconv.Atoi(result.Find("div:nth-child(11) > span > font:nth-child(2) > b").Text())
		magnet, _ := result.Find("div:nth-child(5) > a:nth-child(2)").Attr("href")

		source := models.Source{
			From:     "TorrentGalaxy",
			Title:    title,
			URL:      Site + URL, // Assuming 'Site' is declared somewhere
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(size),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("TorrentGalaxy: [%d] Amount of results: %d", page, len(sources))
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
	modifiedQuery := strings.ReplaceAll(query, " ", "+")
	return modifiedQuery
}
