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

	// 👇 Play Store BLOCKS requests without User-Agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// FAKE as real Chrome Browser
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "+
			"(KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch Play Store page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Play Store returned status %d", res.StatusCode)
	}

	return goquery.NewDocumentFromReader(res.Body)
}
