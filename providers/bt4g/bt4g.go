package bt4g

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
	Name = "Bt4g"
	Site = "https://bt4gprx.com"
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
		// Extract information from each search result item
		title := result.Find("h5 a").Text()
		URL, _ := result.Find("h5 a").Attr("href")

		// Extract other relevant information
		filesizeStr := result.Find("b.cpill").Text()
		filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))

		seedersStr := result.Find("b#seeders").Text()
		seeders, _ := strconv.Atoi(seedersStr)

		leechersStr := result.Find("b#leechers").Text()
		leechers, _ := strconv.Atoi(leechersStr)

		magnet, _ := result.Find("h5 a").Attr("href")

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
