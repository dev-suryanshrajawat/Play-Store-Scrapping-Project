package scraper

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Timeout = 3 seconds (PERFORMANCE NFR)
var httpClient = &http.Client{
	Timeout: 3 * time.Second,
}

func FetchPlayStoreHTML(pkg string) (*goquery.Document, error) {

	if !strings.Contains(pkg, ".") {
		return nil, fmt.Errorf("invalid package name, use format like com.whatsapp")
	}

	url := fmt.Sprintf(
		"https://play.google.com/store/apps/details?id=%s&hl=en_US&gl=US",
		pkg,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request build failed: %v", err)
	}

	// PERFORMANCE BOOST: Real Browser Headers
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Mobile Safari/537.36")

	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Referer", "https://www.google.com/")

	// PERFORMANCE: Persistent client reused every time
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Play Store page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("play store returned status %d", res.StatusCode)
	}

	return goquery.NewDocumentFromReader(res.Body)
}
