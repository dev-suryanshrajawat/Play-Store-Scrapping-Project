package scraper

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

// FetchPlayStoreHTML fetches the Play Store HTML page for a package
func FetchPlayStoreHTML(pkg string) (*goquery.Document, error) {
	url := fmt.Sprintf("https://play.google.com/store/apps/details?id=%s&hl=en&gl=us", pkg)
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Play Store page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("app not found on Play Store")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Play Store HTML")
	}

	return doc, nil
}
