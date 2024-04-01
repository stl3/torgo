package audiobookbay

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"

	"github.com/stl3/torgo/models"
)

const (
	Name = "Audiobookbay"
	Site = "https://audiobookbay.is"
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
		// All:       "/%d/%v/",
		Audiobook: "/page/%v/?s=%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	modifiedQuery := modifyQuery(query)
	logrus.Infof("Modified query before changes: %s", modifiedQuery)
	// modifiedQuery = string(query[0]) + "/" + modifiedQuery
	// logrus.Infof("Modified query afer changes: %s", modifiedQuery)
	results, err := provider.Query(modifiedQuery, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	// Replace "%2F" with "/"
	// surl = strings.ReplaceAll(surl, "%2F", "/")
	newSurl := rearrangeURL(surl)
	newSurl = strings.ReplaceAll(newSurl, "?s=page/", "?s=")
	logrus.Infof("Surl: [%s]...\n", surl)
	logrus.Infof("newSurl: [%s]...\n", newSurl)
	// Log or display the full URL before making the request
	logrus.Infof("Audiobookbay: [%d] Requesting URL: %s\n", page, newSurl)

	logrus.Infof("Audiobookbay: [%d] Extracting results...\n", page)
	// _, html, err := request.Get(nil, strings.ReplaceAll(surl, "/", "%2F"), nil)
	// headers := map[string]string{
	// 	"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	// }
	// _, html, err := request.Get(nil, surl, headers)
	// logrus.Infof("html: [%s]...\n", html)
	// if err != nil {
	// 	logrus.Errorln(fmt.Sprintf("Audiobookbay: [%d]", page), err)
	// 	wg.Done()
	// 	return
	// }

	// Define custom headers
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}

	// Create a new HTTP client with default settings
	client := &http.Client{}

	// Create a new HTTP request with custom headers
	req, err := http.NewRequest("GET", surl, nil)
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Perform the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return
	}
	defer resp.Body.Close()

	// Check if the response is a redirect
	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
		// Get the new location from the response header
		newURL := resp.Header.Get("Location")
		fmt.Println("Resource moved permanently to:", newURL)

		// Follow the redirect by making another request to the new location
		extractor(newURL, page, results, wg)
		return
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	logrus.Infof("body: [%s]...\n", body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Process the response body
	html := string(body)
	// logrus.Infof("html: [%s]...\n", html)
	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	// resultsContainer := doc.Find("#content > div.fill-table > table > tbody > tr")
	// resultsContainer.Each(func(_ int, result *goquery.Selection) {
	// 	// Extract information from each search result item
	// 	title := result.Find("td.n a").Text()
	resultsContainer := doc.Find("div.page")
	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		// Extract information from each search result item
		title := result.Find("div.post:nth-child(6) > div:nth-child(1) > h2:nth-child(1) > a:nth-child(1)").Text()

		// table := doc.Find("div.page")
		// table.Find("div.post").Each(func(i int, tr *goquery.Selection) {
		// 	// title
		// 	title := tr.Find("div.post:nth-child(6) > div:nth-child(1) > h2:nth-child(1) > a:nth-child(1)").Text()

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
			From:     "Audiobookbay",
			Title:    title,
			URL:      Site + URL, // Assuming 'Site' is declared somewhere
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(size),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("Audiobookbay: [%d] Amount of results: %d", page, len(sources))
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

// func rearrangeURL(surl string) string {
// 	// Parse the URL string
// 	parsedURL, err := url.Parse(surl)
// 	if err != nil {
// 		fmt.Println("Error parsing URL:", err)
// 		return ""
// 	}

// 	// Extract the path components
// 	pathComponents := strings.Split(parsedURL.Path, "/")
// 	// Get the last component as the string
// 	queryString := pathComponents[len(pathComponents)-1]
// 	// Get the second-to-last component as the integer
// 	pageNumber, err := strconv.Atoi(pathComponents[len(pathComponents)-2])
// 	if err != nil {
// 		fmt.Println("Error converting page number:", err)
// 		return ""
// 	}

// 	// Rearrange the components
// 	newPath := fmt.Sprintf("/page/%d/?s=%s", pageNumber, queryString)
// 	parsedURL.Path = newPath

// 	// Return the modified URL string
// 	return parsedURL.String()
// }

func rearrangeURL(surl string) string {
	// Parse the URL string
	parsedURL, err := url.Parse(surl)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return ""
	}

	// Parse the query parameters
	queryParams, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		fmt.Println("Error parsing query parameters:", err)
		return ""
	}

	// Extract the page number and query string
	pageNumberStr := queryParams.Get("s")
	pageNumber, err := strconv.Atoi(pageNumberStr)
	if err != nil {
		fmt.Println("Error converting page number:", err)
		return ""
	}
	queryString := strings.Trim(parsedURL.Path, "/")

	// Rearrange the components
	newPath := fmt.Sprintf("/page/%d/", pageNumber)
	newQuery := fmt.Sprintf("s=%s", queryString)
	parsedURL.Path = newPath
	parsedURL.RawQuery = newQuery

	// Return the modified URL string
	return parsedURL.String()
}
