package bt4g

import (
	"fmt"
	"net/http"
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
	Name = "Bt4g"
	Site = "https://bt4gprx.com"
)

type provider struct {
	models.Provider
}

func New() models.ProviderInterface {
	provider := &provider{}
	provider.Name = Name
	provider.Site = Site
	provider.Categories = models.Categories{
		All: "/search?q=%v&category=all&orderby=seeders&p=%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	logrus.Infof("Bt4g: [%d] Extracting results...\n", page)

	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("Bt4g: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("div.col.s12 > div")

	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		title := result.Find("h5 a").Text()
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
		URL, _ := result.Find("h5 a").Attr("href")
		logrus.Infof("URL: %s", URL)

		newURL := "https://bt4gprx.com" + URL
		logrus.Infof("newURL: %s", newURL)

		hash, err := getHashFromURL(newURL)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Construct the magnet URI
		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s", hash)

		filesizeStr := result.Find("b.cpill").Text()
		filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))

		seedersStr := result.Find("b#seeders").Text()
		seeders, _ := strconv.Atoi(seedersStr)

		leechersStr := result.Find("b#leechers").Text()
		leechers, _ := strconv.Atoi(leechersStr)

		source := models.Source{
			From:     "Bt4g",
			Title:    title,
			URL:      Site + URL,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("Bt4g: [%d] Amount of results: %d", page, len(sources))
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

func getHashFromURL(url string) (string, error) {
	// Make a GET request to the URL
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// Parse the HTML response
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return "", err
	}

	// Extract the href attribute value from the specified selector
	href := document.Find(".s12 > table:nth-child(3) > tbody:nth-child(2) > tr:nth-child(1) > th:nth-child(1) > a:nth-child(1)").AttrOr("href", "")
	if href == "" {
		return "", fmt.Errorf("href attribute not found")
	}
	href1, err := ExtractMagnetHash(href)
	if err != nil {
		return "", err
	}
	return href1, nil
}

func ExtractMagnetHash(href string) (string, error) {
	// Regular expression to match the magnet hash
	re := regexp.MustCompile(`\/hash\/([a-fA-F0-9]+)`)

	// Find submatches
	matches := re.FindStringSubmatch(href)
	if len(matches) < 2 {
		return "", fmt.Errorf("unable to extract magnet hash from href")
	}

	// Extract and return the magnet hash
	magnetHash := matches[1]
	return magnetHash, nil
}
