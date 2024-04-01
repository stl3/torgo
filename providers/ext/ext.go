package ext

import (
	"fmt"
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

	"github.com/stl3/torrodle/config"
	"github.com/stl3/torrodle/models"
	// "github.com/stl3/torrodle/request"
)

// var configurations config.TorrodleConfig

func init() {
	// Load the configuration
	u, _ := user.Current()
	home := u.HomeDir
	configFile := filepath.Join(home, ".torgo.json")

	configurations, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Println("Error loading config:", err)
		configurations = config.TorrodleConfig{}
	}
	// fmt.Printf("Loaded configuration: %+v\n", configurations)
	logrus.Debugf("Loaded configuration: %+v\n", configurations)
}

const (
	Name = "ext"
	Site = "https://ext.to"
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
		All:   "/search/%v/%d/",
		Movie: "/search/%v/%d/?c=movies",
		TV:    "/search/%v/%d/?c=tv",
		Porn:  "/search/%v/%d/?c=xxx",
		// cookie
		// "cf_chl_3=46f7245221c3a26"
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

// func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {

// 	logrus.Infof("Ext: [%d] Extracting results...\n", page)
// 	_, html, err := request.Get(nil, surl, nil)
// 	if err != nil {
// 		logrus.Errorln(fmt.Sprintf("ext: [%d]", page), err)
// 		wg.Done()
// 		return
// 	}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	// surl = removeNumberedStrings(surl)

	// Log or display the full URL before making the request
	logrus.Infof("Ext: [%d] Requesting URL: %s\n", page, surl)
	logrus.Infof("Ext: [%d] Extracting results...\n", page)

	client := resty.New()

	// // Create cookies
	// cookie1 := &http.Cookie{
	// 	Name:  "PHPSESSID",
	// 	Value: configurations.Ext_cookie,
	// 	// Value: "cfadf97973e6b01",
	// }

	// cookie2 := &http.Cookie{
	// 	Name:  "cf_chl_rc_m",
	// 	Value: "2",
	// }
	// // Add cookies to the request
	// client.SetCookies([]*http.Cookie{cookie1, cookie2})

	// Set the cookie
	client.SetHeader("Cookie", "cf_chl_3=e69362e454d9cfe")
	// Make the request
	resp, err := client.R().Get(surl)

	if err != nil {
		logrus.Errorln(fmt.Sprintf("Ext: [%d]", page), err)
		wg.Done()
		return
	}

	// // Make the request
	// resp, err := client.R().
	// 	SetResult(&struct {
	// 		// Define the structure of the expected response
	// 		// Replace with the actual fields you expect in the response
	// 		// For example, if the response is JSON, define the JSON structure here.
	// 		Field1 string `json:"field1"`
	// 		Field2 int    `json:"field2"`
	// 		// Add more fields as needed
	// 	}{}).
	// 	Get(surl)

	// if err != nil {
	// 	logrus.Errorln(fmt.Sprintf("Ext: [%d]", page), err)
	// 	wg.Done()
	// 	return
	// }

	var sources []models.Source
	html := resp.String()
	fmt.Print(html)
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("tbody > tr")
	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		title := result.Find("td:nth-child(1) > div:nth-child(1) > a:nth-child(2)").Text()
		fmt.Print(title)
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

		filesizeStr := result.Find("td:nth-child(3)").Text()
		filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))
		// date := result.Find("td[title^='20']").Text()
		seeders, _ := strconv.Atoi(result.Find("td:nth-last-child(3)").Text())
		leechers, _ := strconv.Atoi(result.Find("td:nth-last-child(2)").Text())
		magnet, _ := result.Find("td.text-wrap a").Attr("href")

		// if err != nil {
		// 	// log.Println("Error converting sizeStr to int:", err)
		// 	log.Println("Size string:", filesizeStr)
		// }

		source := models.Source{
			From:     "ext",
			Title:    title,
			URL:      surl,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
			// You may add other fields like category, subcategory, date as needed
		}
		sources = append(sources, source)
	})

	logrus.Debugf("Ext: [%d] Amount of results: %d", page, len(sources))
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
