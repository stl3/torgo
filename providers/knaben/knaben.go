package knaben

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
	Name = "knaben"
	Site = "https://knaben.eu"
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
		All:   "/search/%v/0/%d/seeders",
		Movie: "/search/%v/3000000/%d/seeders",
		TV:    "/search/%v/2000000/%d/seeders",
		Porn:  "/search/%v/5000000/%d/seeders",
	}
	return provider
}

func (provider *provider) Search(query string, count int, categoryURL models.CategoryURL) ([]models.Source, error) {
	results, err := provider.Query(query, categoryURL, count, 50, 1, extractor)
	return results, err
}

func extractor(surl string, page int, results *[]models.Source, wg *sync.WaitGroup) {

	logrus.Infof("knaben: [%d] Extracting results...\n", page)
	_, html, err := request.Get(nil, surl, nil)
	if err != nil {
		logrus.Errorln(fmt.Sprintf("knaben: [%d]", page), err)
		wg.Done()
		return
	}

	var sources []models.Source
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	resultsContainer := doc.Find("tbody tr")

	resultsContainer.Each(func(_ int, result *goquery.Selection) {
		title := result.Find("td.text-wrap a").AttrOr("title", "")
		URL, _ := result.Find("td.text-wrap a").Attr("href")
		filesizeStr := result.Find("td:nth-child(3)").Text()
		filesize, _ := humanize.ParseBytes(strings.TrimSpace(filesizeStr))
		seedersStr := result.Find("td").Eq(4).Text()
		seeders, _ := strconv.Atoi(seedersStr)
		leechersStr := result.Find("td").Eq(5).Text()
		leechers, _ := strconv.Atoi(leechersStr)

		source := models.Source{
			From:     "knaben",
			Title:    title,
			URL:      Site + URL,
			Seeders:  seeders,
			Leechers: leechers,
			FileSize: int64(filesize),
		}
		sources = append(sources, source)
	})

	logrus.Debugf("knaben: [%d] Amount of results: %d", page, len(sources))
	*results = append(*results, sources...)
	wg.Done()
}
