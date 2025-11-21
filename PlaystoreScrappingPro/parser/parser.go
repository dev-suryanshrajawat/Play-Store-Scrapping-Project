package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// App struct to store parsed data
type App struct {
	AppName          string   `json:"appName"`
	Title            string   `json:"title"`
	Icon             string   `json:"icon"`
	Developer        string   `json:"developer"`
	DeveloperEmail   string   `json:"developerEmail"`
	DeveloperWebsite string   `json:"developerWebsite"`
	Category         string   `json:"genre"`
	Rating           string   `json:"rating"`
	RatingCount      string   `json:"ratingCount"`
	Installs         string   `json:"installs"`
	Free             bool     `json:"free"`
	AdSupported      bool     `json:"adSupported"`
	InAppPurchase    bool     `json:"InAppPurchase"`
	LastUpdated      string   `json:"updated"`
	CurrentVersion   string   `json:"version"`
	AndroidVersion   string   `json:"androidVersion"`
	ShortDesc        string   `json:"summary"`
	Description      string   `json:"description"`
	Screenshots      []string `json:"screenshots"`
}

// ParsePlayStoreHTML extracts app info from goquery.Document with robust fallbacks
func ParsePlayStoreHTML(doc *goquery.Document) (*App, error) {
	app := &App{}

	//Try JSON-LD structured data (preferred)
	// --- JSON-LD extraction (MOST RELIABLE) ---
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())

		// We only want the SoftwareApplication JSON-LD block
		if !strings.Contains(text, "SoftwareApplication") {
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(text), &data); err != nil {
			return
		}

		// Title
		if v := fmt.Sprint(data["name"]); v != "" && v != "<nil>" {
			app.Title = v
		}

		// Icon
		if v := fmt.Sprint(data["image"]); v != "" && v != "<nil>" {
			app.Icon = v
		}

		// AppName
		if v := fmt.Sprint(data["url"]); v != "" && v != "<nil>" {
			app.AppName = v
		}

		// Description
		if v := fmt.Sprint(data["description"]); v != "" && v != "<nil>" {
			app.Description = v
		}

		// Category
		if v := fmt.Sprint(data["applicationCategory"]); v != "" && v != "<nil>" {
			app.Category = v
		}

		// Developer Name
		if author, ok := data["author"].(map[string]interface{}); ok {
			if v := fmt.Sprint(author["name"]); v != "" {
				app.Developer = v
			}
		}

		// ⭐ RATING
		if agg, ok := data["aggregateRating"].(map[string]interface{}); ok {

			// RatingValue
			if rv, ok := agg["ratingValue"]; ok {
				app.Rating = fmt.Sprint(rv)
			}

			// RatingCount
			if rc, ok := agg["ratingCount"]; ok {
				app.RatingCount = fmt.Sprint(rc)
			}

		}
	})

	// --- FALLBACK RATING (HTML aria-label method — SUPER RELIABLE) ---

	// Rating

	label := doc.Find("div[aria-label^='Rated']").AttrOr("aria-label", "")
	// example: “Rated 4.5 stars out of five”
	if label != "" {
		parts := strings.Fields(label)
		if len(parts) > 1 {
			app.Rating = parts[1] // second word is rating
		}
	}
	if app.Rating == "" {
		if f, err := strconv.ParseFloat(app.Rating, 64); err == nil {
			app.Rating = fmt.Sprintf("%.1f", f)
		}
	}

	// RatingCount
	// after setting app.RatingCount from JSON-LD or aria-label
	ratingCount := strings.TrimSpace(
		doc.Find("div.g1rdde").Text(),
	)
	if ratingCount != "" {
		// Extract only numbers from "1,234 ratings"
		re := regexp.MustCompile(`[\d.,]+[KM]?`)
		ratingCount = re.FindString(ratingCount)
		app.RatingCount = ratingCount
	}

	if app.RatingCount == "" {
		app.RatingCount = "0"
	}

	// url contains canonical url (with id param)
	if app.AppName == "" {
		app.AppName = strings.TrimSpace(doc.Find(`meta[property="og:url"]`).AttrOr("content", ""))
	}
	// icon fallback from og:image
	if app.Icon == "" {
		app.Icon = strings.TrimSpace(doc.Find(`meta[property="og:image"]`).AttrOr("content", ""))
	}
	// short description / meta description
	if app.ShortDesc == "" {
		app.ShortDesc = strings.TrimSpace(doc.Find(`meta[name="description"]`).AttrOr("content", ""))
	}

	// Full Description
	app.Description = strings.Join(strings.Fields(strings.TrimSpace(
		doc.Find("div[jsname='sngebd']").First().Text(),
	)), " ")

	if app.Description == "" {
		app.Description = strings.Join(strings.Fields(strings.TrimSpace(
			doc.Find("div[data-g-id='description']").First().Text(),
		)), " ")
	}

	//Developer website
	if app.Developer == "" {
		app.Developer = strings.TrimSpace(doc.Find("a.hrTbp.R8zArc").First().Text())
	}
	// developer email (mailto:)
	if app.DeveloperEmail == "" {
		if mail := doc.Find("a[href^='mailto:']").AttrOr("href", ""); mail != "" {
			app.DeveloperEmail = strings.TrimSpace(mail)
		}
	}
	// developer website: look for links under developer section or any https link with developer pattern
	if app.DeveloperWebsite == "" {
		// prefer explicit developer website link
		if href := doc.Find(`a[href*="developer"]`).First().AttrOr("href", ""); href != "" {
			app.DeveloperWebsite = strings.TrimSpace(href)
		} else {
			// generic fallback: first https link in page that is not play.google
			doc.Find("a[href^='http']").EachWithBreak(func(i int, s *goquery.Selection) bool {
				h := s.AttrOr("href", "")
				if h == "" {
					return true
				}
				if strings.Contains(h, "play.google.com") {
					return true // skip
				}
				app.DeveloperWebsite = strings.TrimSpace(h)
				return false
			})
		}
	}

	// ---------------- DETAILS SECTION (updated & robust) ----------------
	doc.Find("div.VfPpkd-A7Ei6b, div.VfPpkd-qRZikd, div.UCQdA").Each(func(i int, s *goquery.Selection) {

		// Try multiple label selectors (Google keeps changing)
		label := strings.TrimSpace(s.Find("div.BgcNfc").Text())
		if label == "" {
			label = strings.TrimSpace(s.Find("div.wVqUob").Text())
		}
		if label == "" {
			label = strings.TrimSpace(s.Find("div.qQjadf").Text())
		}
		if label == "" {
			// sometimes label is direct child text
			label = strings.TrimSpace(s.Find("span").First().Text())
		}

		// Try multiple value selectors (value may be in a sibling/span/div)
		value := strings.TrimSpace(s.Find("span.htlgb").Text())
		if value == "" {
			value = strings.TrimSpace(s.Find("div.reAt0").Text())
		}
		if value == "" {
			value = strings.TrimSpace(s.Find("div.Uc9Gjf").Text())
		}
		if value == "" {
			// sometimes the value is the last span in the block
			value = strings.TrimSpace(s.Find("span").Last().Text())
		}
		// normalize label to lower for switching
		l := strings.ToLower(strings.TrimSpace(label))

		switch {
		case strings.Contains(l, "updated"):
			if app.LastUpdated == "" {
				app.LastUpdated = value
			}
			//Current Version And Android Version
		case strings.Contains(l, "current version") || strings.Contains(l, "version"):
			if app.CurrentVersion == "" {
				app.CurrentVersion = value
			}
		case strings.Contains(l, "requires android") || strings.Contains(l, "requires"):
			if app.AndroidVersion == "" {
				app.AndroidVersion = value
			}
		case strings.Contains(l, "installs") || strings.Contains(l, "downloads"):
			if app.Installs == "" {
				app.Installs = value
			}
		}
	})

	// Default placeholders
	if app.CurrentVersion == "" {
		app.CurrentVersion = "N.A"
	}
	if app.AndroidVersion == "" {
		app.AndroidVersion = "N.A"
	}
	if app.Installs == "" {
		app.Installs = "N.A"
	}

	// ---------- Fallback: in case the above misses ----------
	// Common "label + value" fallback using adjacent sibling selectors
	if app.LastUpdated == "" {
		app.LastUpdated = strings.TrimSpace(doc.Find("div:contains('Updated on') + div, div:contains('Updated') + div").First().Text())
	}
	if app.CurrentVersion == "" && app.CurrentVersion == "N.A" {
		app.CurrentVersion = strings.TrimSpace(doc.Find("div:contains('Current Version') + div, div:contains('Version') + div").First().Text())
	}
	if app.AndroidVersion == "" && app.AndroidVersion == "N.A" {
		app.AndroidVersion = strings.TrimSpace(doc.Find("div:contains('Requires Android') + div, div:contains('Requires') + div").First().Text())
	}

	// Try alternate selectors for installs (new classes / common locations)
	if app.Installs == "" || app.Installs == "N.A" {
		// new-ish class that often contains installs
		if txt := strings.TrimSpace(doc.Find("div.Uc9Gjf, div.reAt0, div.VfPpkd-A7Ei6b").Last().Text()); txt != "" {
			// sometimes these blocks include label+value; try to extract only the numeric part
			app.Installs = strings.TrimSpace(txt)
		}

		// some pages put installs in small spans under developer section
		if app.Installs == "" || app.Installs == "N.A" {
			if txt := strings.TrimSpace(doc.Find("div.wVqUob span").Last().Text()); txt != "" {
				app.Installs = txt
			}
		}
	}

	// If still empty, scan <script> tags for numDownloads or similar keys
	if app.Installs == "" || app.Installs == "N.A" {
		found := ""
		doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
			t := s.Text()
			lower := strings.ToLower(t)
			if strings.Contains(lower, "numdownloads") || strings.Contains(lower, "num_downloads") || strings.Contains(lower, "downloads") {
				// attempt simple extraction using common patterns
				// look for "numDownloads": "1000000+" or "numDownloads":"1,000,000+"
				re := regexp.MustCompile(`(?i)("numDownloads"\s*[:=]\s*"([^"]+)")|("num_downloads"\s*[:=]\s*"([^"]+)")|((?:["']downloads["']\s*[:=]\s*")([^"]+)")`)
				if m := re.FindStringSubmatch(t); m != nil {
					// find the first non-empty capture group that corresponds to the value
					for j := 2; j < len(m); j++ {
						if m[j] != "" {
							found = strings.TrimSpace(m[j])
							break
						}
					}
					if found != "" {
						app.Installs = found
						return false // break
					}
				}
				// as a looser fallback, try to find any "downloads" phrase nearby
				loose := regexp.MustCompile(`(?i)[\d,\.]+(?:\+| ?[KkMmBb]| ?cr| ?lakh)?\s*(?:downloads|installs)`)
				if mm := loose.FindString(t); mm != "" {
					app.Installs = strings.TrimSpace(mm)
					return false
				}
			}
			return true
		})
		if found != "" {
			app.Installs = found
		}
	}

	// Final wide-text regex pass over page text (most reliable for static HTML text like "50M+ downloads")
	if app.Installs == "" || app.Installs == "N.A" {
		pageText := strings.ToLower(doc.Text())
		// regex: capture patterns like "50M+ downloads", "1,000,000+ installs", "5 cr+ downloads"
		reDownloads := regexp.MustCompile(`(?i)([\d\.,]+(?:\+| ?[kKmM]| ?cr| ?lakh)?\+?)\s*(?:downloads|installs|downloads\))`)
		if m := reDownloads.FindStringSubmatch(pageText); m != nil || len(m) > 1 {
			app.Installs = strings.TrimSpace(m[0])
		}
	}

	// final normalization: if installs is still empty set N.A
	if app.Installs == "" {
		app.Installs = "N.A"
	}

	//Screenshots
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if len(app.Screenshots) >= 5 {
			return
		}
		// check src and srcset
		src, _ := s.Attr("src")
		if src == "" {
			src = s.AttrOr("data-src", "")
		}
		if src == "" {
			// check srcset: take the first url
			if ss := s.AttrOr("srcset", ""); ss != "" {
				parts := strings.Split(ss, ",")
				if len(parts) > 0 {
					first := strings.TrimSpace(parts[0])
					urlPart := strings.Fields(first)[0]
					if urlPart != "" {
						src = urlPart
					}
				}
			}
		}
		if src == "" {
			return
		}
		if strings.Contains(src, "play-lh.googleusercontent.com") || strings.Contains(src, "play-lh") {
			// avoid duplicates
			already := false
			for _, ex := range app.Screenshots {
				if ex == src {
					already = true
					break
				}
			}
			if !already {
				app.Screenshots = append(app.Screenshots, src)
			}
		}
	})

	//Detect InAppPurchases
	pageText := strings.ToLower(doc.Text())
	if strings.Contains(pageText, "contains ads") || strings.Contains(pageText, "contains advertising") {
		app.AdSupported = true
	}
	if strings.Contains(pageText, "in-app purchases") || strings.Contains(pageText, "in-app billing") {
		app.InAppPurchase = true
	}

	//Free Apps
	price := strings.TrimSpace(doc.Find(`meta[itemprop="price"]`).AttrOr("content", ""))
	if price == "" {
		// fallback: check page text for "free"
		app.Free = strings.Contains(strings.ToLower(doc.Text()), "free")
	} else {
		app.Free = (price == "0" || strings.EqualFold(price, "free"))
	}

	if app.Title == "" {
		app.Title = strings.TrimSpace(doc.Find("h1 span").First().Text())
	}
	if app.Icon == "" {

		app.Icon = strings.TrimSpace(doc.Find("img.T75of").AttrOr("src", ""))
		if app.Icon == "" {
			app.Icon = strings.TrimSpace(doc.Find("img[itemprop='image']").AttrOr("src", ""))
		}
	}
	if app.Category == "" {
		app.Category = strings.TrimSpace(doc.Find("a[itemprop='genre']").First().Text())
	}
	if app.ShortDesc == "" {
		app.ShortDesc = strings.TrimSpace(doc.Find("meta[name='description']").AttrOr("content", ""))
	}
	if app.Description == "" {
		// final fallback to some long containers
		app.Description = strings.TrimSpace(doc.Find("div[jsname='sngebd']").Text())
	}
	if app.AppName == "" {
		// try extracting id from og:url or canonical link
		if u := doc.Find(`link[rel="canonical"]`).AttrOr("href", ""); u != "" {
			app.AppName = u
		} else if u := doc.Find(`meta[property="og:url"]`).AttrOr("content", ""); u != "" {
			app.AppName = u
		}
	}

	if app.Category == "" {
		app.Category = "N/A"
	}
	if app.Installs == "" {
		app.Installs = "N/A"
	}
	if app.Description == "" {
		app.Description = "No description available"
	}
	if app.ShortDesc == "" {
		app.ShortDesc = app.Description
	}

	if app.Title == "" {
		return nil, fmt.Errorf("app not found on Play Store")
	}

	return app, nil
}


