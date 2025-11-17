package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/dev-suryanshrajawat/Play-Store-Scrapping-Project/PlaystoreScrappingPro/output"
	"github.com/dev-suryanshrajawat/Play-Store-Scrapping-Project/PlaystoreScrappingPro/parser"
	"github.com/dev-suryanshrajawat/Play-Store-Scrapping-Project/PlaystoreScrappingPro/scraper"

	"github.com/gin-gonic/gin"

	tollbooth "github.com/didip/tollbooth/v7"
	tollbooth_gin "github.com/didip/tollbooth_gin"
)

///////////////////////////////////////////////////////////////////////////////
// SECURITY — sanitize and validate package name
///////////////////////////////////////////////////////////////////////////////

func sanitizePackage(pkg string) (string, error) {

	pkg = strings.TrimSpace(pkg)
	pkg = strings.ToLower(pkg)

	if pkg == "" {
		return "", fmt.Errorf("package name is required")
	}

	if len(pkg) > 60 {
		return "", fmt.Errorf("package name too long")
	}

	if !strings.Contains(pkg, ".") {
		return "", fmt.Errorf("invalid package format (use com.example.app)")
	}

	if strings.ContainsAny(pkg, "/\\?*&=<>'\"{}()[]|;: ") {
		return "", fmt.Errorf("unsafe characters detected")
	}

	for _, c := range pkg {
		if !(c >= 'a' && c <= 'z') &&
			!(c >= '0' && c <= '9') &&
			c != '.' {
			return "", fmt.Errorf("invalid character in package name")
		}
	}

	return pkg, nil
}

///////////////////////////////////////////////////////////////////////////////
// SCALABLE CACHE — Thread-Safe with Expiry
///////////////////////////////////////////////////////////////////////////////

type CacheEntry struct {
	Data      *parser.App
	Timestamp int64
}

var Cache = make(map[string]CacheEntry)
var cacheLock = &sync.RWMutex{}

const CacheTTL = 6 * 60 * 60 // 6 hours

func getFromCache(pkg string) (*parser.App, bool) {
	cacheLock.RLock()
	entry, found := Cache[pkg]
	cacheLock.RUnlock()

	if !found {
		return nil, false
	}

	if time.Now().Unix()-entry.Timestamp > CacheTTL {
		// expired → delete
		cacheLock.Lock()
		delete(Cache, pkg)
		cacheLock.Unlock()
		return nil, false
	}

	return entry.Data, true
}

func saveToCache(pkg string, app *parser.App) {
	cacheLock.Lock()
	Cache[pkg] = CacheEntry{
		Data:      app,
		Timestamp: time.Now().Unix(),
	}
	cacheLock.Unlock()
}

///////////////////////////////////////////////////////////////////////////////
// MAIN SERVER
///////////////////////////////////////////////////////////////////////////////

func main() {

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	//-----------------------------------------------------------------------
	// RATE LIMITING (5 req/sec, burst 10)
	//-----------------------------------------------------------------------
	limiter := tollbooth.NewLimiter(5, nil)
	limiter.SetBurst(10)
	limiter.SetIPLookups([]string{
		"RemoteAddr",
		"X-Forwarded-For",
		"X-Real-IP",
	})
	limiter.SetMessageContentType("application/json")
	limiter.SetMessage(`{"found": false, "error": "Rate limit exceeded"}`)

	//-----------------------------------------------------------------------
	// HOME PAGE
	//-----------------------------------------------------------------------
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	//-----------------------------------------------------------------------
	// APP INFO ROUTE — security + caching + retry + scalability
	//-----------------------------------------------------------------------
	r.GET("/app-info", tollbooth_gin.LimitHandler(limiter), func(c *gin.Context) {

		raw := c.Query("package")

		// SECURITY
		pkg, err := sanitizePackage(raw)
		if err != nil {
			output.ShowErrorPage(c, err.Error())
			return
		}

		// CACHE CHECK
		if app, ok := getFromCache(pkg); ok {
			fmt.Println("CACHE HIT:", pkg)
			output.ShowAppInfo(c, app)
			return
		}

		// RETRY SCRAPER (3 attempts)
		var doc *goquery.Document
		for retry := 1; retry <= 3; retry++ {
			doc, err = scraper.FetchPlayStoreHTML(pkg)
			if err == nil {
				break
			}
			fmt.Println("Retry:", retry, "for", pkg)
			time.Sleep(time.Second)
		}

		if err != nil {
			output.ShowErrorPage(c, "Failed to reach Google Play. Try again.")
			return
		}

		// PARSE APP
		app, err := parser.ParsePlayStoreHTML(doc)
		if err != nil {
			output.ShowErrorPage(c, err.Error())
			return
		}

		// SAVE TO CACHE (thread-safe)
		saveToCache(pkg, app)
		fmt.Println("CACHE SAVED:", pkg)

		// DISPLAY RESULT
		output.ShowAppInfo(c, app)
	})

	r.Run(":8000")
}

