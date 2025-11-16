package scraper

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func FetchPlayStoreHTML(pkg string) (*goquery.Document, error) {

	if !strings.Contains(pkg, ".") {
		return nil, fmt.Errorf("Invalid package name. Use format like com.whatsapp")
	}

	url := fmt.Sprintf("https://play.google.com/store/apps/details?id=%s&hl=en&gl=us", pkg)
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Play Store page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("app not found on Play Store")
	}

	return goquery.NewDocumentFromReader(res.Body)
}
