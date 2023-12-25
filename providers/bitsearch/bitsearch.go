package bitsearch

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"

	"github.com/stl3/torrodle/models"
	"github.com/stl3/torrodle/request"
)

const (
	Name = "Bitsearch"
	Site = "https://bitsearch.to"
)

// var Site string // Package-level variable

type provider struct {
	models.Provider
}

func New() models.ProviderInterface {
	// var Site string

	provider := &provider{}
	provider.Name = Name
	provider.Site = Site
	provider.Categories = models.Categories{
		// All:   "/search/all/%v/seeds/%d",
		All:   "/search?q=%v&page=%d",
		Movie: "/search?q=%v&page=%d&category=1",
		TV:    "/search?q=%v&page=%d",
		Anime: "/search?q=%v&page=%d",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {
	// func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup, Site string) { // Add Site as a parameter

	logrus.Infof("Bitsearch: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("Bitsearch: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("li.card.search-result")

	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		// Extract information from each search result item
		title := result.Find("h5.title a").Text()
		URL, _ := result.Find("h5.title a").Attr("href")
		// Extract other relevant information
		filesizeStr := result.Find("div.stats div:nth-child(2)").Text()
		// filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))
		// sizeStr := result.Find("div.stats div:contains('Size')").Text()
		// sizeStr = strings.TrimSpace(strings.TrimPrefix(sizeStr, "Size"))
		filesize, err := humanize.ParseBytes(filesizeStr)
		// if err != nil {
		// 	// Handle the error, e.g., print or log it
		// 	fmt.Println("Error parsing size:", err)
		// } else {
		// 	// Now 'size' contains the size in bytes
		// 	fmt.Println("Size in bytes:", filesize)
		// }

		seedersStr := result.Find("div.stats div:nth-child(3) font").Text()
		// seedersStr := result.Find("div.stats div:contains('Seeder') font").Text()
		// seedersStr := result.Find("div.w3-col.s12.mt-1 > li:nth-child(2) > div.info.px-3.pt-2.pb-3 > div > div > div > div:nth-child(4)").Text()
		seeders, _ := strconv.Atoi(seedersStr)

		leechersStr := result.Find("div.stats div:nth-child(4) font").Text()
		// leechersStr := result.Find("div.stats div:contains('Leecher') font").Text()
		// leechersStr := result.Find("body > main > div.container.mt-2 > div > div.w3-col.s12.mt-1 > li:nth-child(2) > div.info.px-3.pt-2.pb-3 > div > div > div > div:nth-child(4)").Text()
		// leechersStr := result.Find("body > main > div.container.mt-2 > div > div.w3-col.s12.mt-1 > li:nth-child(2) > div.info.px-3.pt-2.pb-3 > div > div > div > div:nth-child(4)").Text()
		leechers, _ := strconv.Atoi(leechersStr)
		// Update leechers extraction to consider the font color
		// leechersStr := result.Find("div.stats div:contains('Leecher') font").Text()
		// leechers, _ := strconv.Atoi(leechersStr)

		if err != nil {
			// Print the leechers string for debugging
			fmt.Println("Error converting leechersStr to int:", err)
			fmt.Println("Leechers string:", leechersStr)
		}
		fmt.Println("Leechers count:", leechers)
		magnet, _ := result.Find("div.links a.dl-magnet").Attr("href")

		source := models.Source{
			From:     "Bitsearch",
			Title:    title,
			URL:      Site + URL,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
			Magnet:   magnet,
		}
		sources = append(sources, source)
	})

	logrus.Debugf("Bitsearch: [%d] Amount of results: %d", page, len(sources))
	*results = append(*results, sources...)
	wg.Done()
}
