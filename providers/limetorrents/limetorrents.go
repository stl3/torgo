package limetorrents

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"

	"github.com/stl3/torrodle/models"
	"github.com/stl3/torrodle/request"
)

const (
	Name = "LimeTorrents"
	// Site = "https://www.limetorrents.info"
	// Site = "https://www.limetorrents.lol"
	// DefaultSite is the default LimeTorrents site URL
	DefaultSite     = "https://www.limetorrents.lol"
	AlternativeSite = "https://www.limetorrents.info"
)

var Site string // Package-level variable

type provider struct {
	models.Provider
}

func checkSiteAvailability(siteURL string) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	_, err := client.Get(siteURL)
	return err == nil
}

func New() models.ProviderInterface {
	// var Site string

	if checkSiteAvailability(DefaultSite) {
		Site = DefaultSite
		// log.Printf("Using site: %s\n", Site)
	} else if checkSiteAvailability(AlternativeSite) {
		Site = AlternativeSite
		// log.Printf("Using site: %s\n", Site)
	} else {
		// Both sites are down, you can handle this case accordingly
		// panic("Both LimeTorrents sites are down")
		// log.Fatal("Both LimeTorrents sites are down")
		Site = DefaultSite
	}

	provider := &provider{}
	provider.Name = Name
	provider.Site = Site
	provider.Categories = models.Categories{
		All:   "/search/all/%v/seeds/%d",
		Movie: "/search/movies/%v/seeds/%d",
		TV:    "/search/tv/%v/seeds/%d",
		Anime: "/search/anime/%v/seeds/%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {

	logrus.Infof("LimeTorrents: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("LimeTorrents: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("table.table2 tbody tr")

	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		titleContainer := result.Find("td.tdleft div.tt-name a:last-child")
		title := titleContainer.Text()
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
		URL, _ := result.Find("td.tdleft > div.tt-name > a:nth-child(2)").Attr("href")
		filesizeStr := result.Find("td.tdnormal:nth-child(3)").Text()
		// filesize, err := humanize.ParseBytes(filesizeStr)
		filesize, _ := humanize.ParseBytes(filesizeStr)
		seedersStr := result.Find("td.tdseed").Text()
		seeders, _ := strconv.Atoi(seedersStr)
		leechersStr := result.Find("td.tdleech").Text()
		leechers, _ := strconv.Atoi(leechersStr)

		// if err != nil {
		// 	// Handle error if conversion fails
		// 	// fmt.Println("Error converting filesizeStr to bytes:", err)
		// 	fmt.Println("Filesize string:", filesizeStr)
		// }
		magnetURL, _ := result.Find("td.tdleft > div.tt-name > a.csprite_dl14").Attr("href")
		re := regexp.MustCompile(`torrent/([^/.]+)\.torrent`)
		matches := re.FindStringSubmatch(magnetURL)
		if len(matches) < 2 {
			// fmt.Println("Error extracting hash from URL")
			return
		}
		hash := matches[1]
		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s", hash)

		source := models.Source{
			From:     "Limetorrents",
			Title:    title,
			URL:      Site + URL,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("LimeTorrents: [%d] Amount of results: %d", page, len(sources))
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
