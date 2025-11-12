package scraper

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// findPackageID searches Play Store for an app name and returns its package ID.
func findPackageID(appName string) (string, error) {
	searchURL := fmt.Sprintf("https://play.google.com/store/search?q=%s&c=apps", appName)

	res, err := http.Get(searchURL)
	if err != nil {
		return "", fmt.Errorf("failed to search Play Store: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("search returned status code %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse search HTML: %v", err)
	}

	var pkg string
	doc.Find("a[href*='/store/apps/details?id=']").EachWithBreak(func(i int, s *goquery.Selection) bool {
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, "details?id=") {
			pkg = strings.Split(strings.Split(href, "id=")[1], "&")[0]
			return false // stop after first match
		}
		return true
	})

	if pkg == "" {
		return "", fmt.Errorf("no package ID found for app '%s'", appName)
	}
	return pkg, nil
}

// FetchPlayStoreHTML fetches Play Store HTML for a given app name (not package).
func FetchPlayStoreHTML(pkg string) (*goquery.Document, error) {
	// Reject if user entered a package ID (e.g. com.whatsapp)
	if strings.Contains(pkg, ".") {
		return nil, fmt.Errorf("No App Found...")
	}

	// Otherwise, search for package ID automatically
	foundPkg, err := findPackageID(pkg)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://play.google.com/store/apps/details?id=%s&hl=en&gl=us", foundPkg)
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Play Store page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("app not found on Play Store (status %d)", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Play Store HTML: %v", err)
	}

	return doc, nil
}
