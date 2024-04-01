package eztv

import (
	"fmt"
	"net/http"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"

	"github.com/stl3/torgo/config"
	"github.com/stl3/torgo/models"
)

// var u, _ = user.Current()
// var home = u.HomeDir

// var configFile = filepath.Join(home, ".torgo.json")

var configurations config.torgoConfig

func init() {
	// Load the configuration
	u, _ := user.Current()
	home := u.HomeDir
	configFile := filepath.Join(home, ".torgo.json")

	configurations, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Println("Error loading config:", err)
		configurations = config.torgoConfig{}
	}
	// fmt.Printf("Loaded configuration: %+v\n", configurations)
	logrus.Debugf("Loaded configuration: %+v\n", configurations)
}

const (
	Name = "eztv"
	Site = "https://eztvx.to"
	// Site = "https://eztv.re"
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
		TV: "/search/%v&%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	modifiedQuery := modifyQuery(query)
	logrus.Infof("Modified query afer changes: %s", modifiedQuery)
	results, err := provider.Query(modifiedQuery, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	surl = removeNumberedStrings(surl)

	// Log or display the full URL before making the request
	logrus.Infof("EZTV: [%d] Requesting URL: %s\n", page, surl)
	logrus.Infof("EZTV: [%d] Extracting results...\n", page)

	client := resty.New()

	// Create cookies
	cookie1 := &http.Cookie{
		Name:  "PHPSESSID",
		Value: configurations.Eztv_cookie,
	}

	cookie2 := &http.Cookie{
		Name:  "layout",
		Value: "def_wlinks",
	}

	// Add cookies to the request
	client.SetCookies([]*http.Cookie{cookie1, cookie2})

	// Make the request
	resp, err := client.R().
		SetResult(&struct {
			// Define the structure of the expected response
			// Replace with the actual fields you expect in the response
			// For example, if the response is JSON, define the JSON structure here.
			Field1 string `json:"field1"`
			Field2 int    `json:"field2"`
			// Add more fields as needed
		}{}).
		Get(surl)

	if err != nil {
		logrus.Errorln(fmt.Sprintf("EZTV: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source

	html := resp.String()
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("tbody tr.forum_header_border")
	resultsContainer.Each(func(_ int, result *goquery.Selection) {

		title := result.Find("td.forum_thread_post > a.epinfo").Text()
		// logrus.Infof("Title: %s", title)

		if containsHTMLEncodedEntities(title) {
			decodedTitle, err := decodeHTMLText(title)
			if err != nil {
				logrus.Errorf("Error decoding HTML text: %v", err)
				// Handle error if necessary
				decodedTitle = title
			}
			logrus.Infof("Decoded Title: %s", decodedTitle)
		}

		URL, _ := result.Find("td.forum_thread_post > a.epinfo").Attr("href")
		// logrus.Infof("URL: %s", URL)

		sizeStr := result.Find("td:nth-child(4)").Text() // Assuming size is in the previous td
		// logrus.Infof("Size: %s", sizeStr)
		size, err := humanize.ParseBytes(sizeStr)

		// seeders, _ := strconv.Atoi(result.Find("td.forum_thread_post_end > font").Text())
		// Remove commas from string, eg "1,201" -> "1201"
		seeders, _ := strconv.Atoi(strings.ReplaceAll(result.Find("td.forum_thread_post_end > font").Text(), ",", ""))
		// logrus.Infof("Seeders: %d", seeders)

		// magnet, _ := result.Find("td.forum_thread_post > a.magnet").Attr("href")
		magnet, _ := result.Find("td:nth-child(3) > a.magnet").Attr("href")
		logrus.Infof("Magnet: %s", magnet)

		if err != nil {
			logrus.Errorf("Error converting sizeStr to int: %v", err)
			logrus.Infof("Size string: %s", sizeStr)
		}

		source := models.Source{
			From:     "EZTV",
			Title:    title,
			URL:      Site + URL, // Assuming 'Site' is declared somewhere
			Seeders:  seeders,
			FileSize: int64(size),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("EZTV: [%d] Amount of results: %d", page, len(sources))
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
	// Replace spaces with "-"
	modifiedQuery := strings.ReplaceAll(query, " ", "-")
	return modifiedQuery
}

func removeNumberedStrings(s string) string {
	for i := 1; i <= 9; i++ {
		s = strings.ReplaceAll(s, fmt.Sprintf("&%d", i), "")
	}
	return s
}
