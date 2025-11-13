package parser

import (
	"encoding/json"
	"fmt"
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
	Rating           float64  `json:"rating"`
	RatingCount      int      `json:"ratingCount"`
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
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Some pages include multiple JSON-LD blocks; find the one for SoftwareApplication
		if !strings.Contains(text, `"SoftwareApplication"`) && !strings.Contains(text, `"@type": "SoftwareApplication"`) {
			return
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(text), &data); err != nil {
			return
		}

		// title, image, url, description, category
		if v := fmt.Sprint(data["name"]); v != "<nil>" && v != "" {
			app.Title = v
		}
		if v := fmt.Sprint(data["image"]); v != "<nil>" && v != "" {
			app.Icon = v
		}
		if v := fmt.Sprint(data["url"]); v != "<nil>" && v != "" {
			app.AppName = v
		}
		if v := fmt.Sprint(data["description"]); v != "<nil>" && v != "" {
			app.Description = v
		}
		if v := fmt.Sprint(data["applicationCategory"]); v != "<nil>" && v != "" {
			app.Category = v
		}

		// author -> developer
		if author, ok := data["author"].(map[string]interface{}); ok {
			if v := fmt.Sprint(author["name"]); v != "<nil>" && v != "" {
				app.Developer = v
			}
		}

		// aggregateRating
		if agg, ok := data["aggregateRating"].(map[string]interface{}); ok {
			if rv, ok := agg["ratingValue"]; ok {
				switch t := rv.(type) {
				case float64:
					app.Rating = t
				case string:
					if p, err := strconv.ParseFloat(t, 64); err == nil {
						app.Rating = p
					}
				}
			}
			if rc, ok := agg["ratingCount"]; ok {
				switch t := rc.(type) {
				case float64:
					app.RatingCount = int(t)
				case string:
					if p, err := strconv.Atoi(t); err == nil {
						app.RatingCount = p
					}
				}
			}
		}
	})

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

	//"Details" section parsing
	// ---------- Details section App (supports old + new Play Store layouts) ----------
	doc.Find("div.hAyfc, div.ClM7O").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find("div.BgcNfc, div.wVqUob").Text())
		value := strings.TrimSpace(s.Find("span.htlgb, div.reAt0").Text())

		switch strings.ToLower(label) {
		case "updated", "updated on":
			if app.LastUpdated == "" {
				app.LastUpdated = value
			}
		case "current version", "version":
			if app.CurrentVersion == "" {
				app.CurrentVersion = value
			}
		case "requires android":
			if app.AndroidVersion == "" {
				app.AndroidVersion = value
			}
		case "installs", "downloads":
			if app.Installs == "" {
				app.Installs = value
			}
		}
	})

	// ---------- Fallback: in case the above misses ----------
	if app.LastUpdated == "" {
		app.LastUpdated = strings.TrimSpace(doc.Find("div:contains('Updated on') + div").First().Text())
	}
	if app.CurrentVersion == "" {
		app.CurrentVersion = strings.TrimSpace(doc.Find("div:contains('Current Version') + div").First().Text())
	}
	if app.AndroidVersion == "" {
		app.AndroidVersion = strings.TrimSpace(doc.Find("div:contains('Requires Android') + div").First().Text())
	}
	//Try alternate selectors for installs
	if app.Installs == "" {
		// commonly visible under div.wVqUob or span
		if txt := strings.TrimSpace(doc.Find("div.wVqUob span").Last().Text()); txt != "" {
			app.Installs = txt
		}
		// some pages put installs in meta tags or in other divs
		if app.Installs == "" {
			doc.Find("div:contains('Downloads'), div:contains('Installs')").Each(func(i int, s *goquery.Selection) {
				if app.Installs == "" {
					app.Installs = strings.TrimSpace(s.Text())
					fmt.Println(app.Installs)
				}
			})
		}
	}

	//Rating and rating count fallback (if JSON-LD didn't provide)
	if app.Rating == 0.0 {
		// try element with rating value
		if r := strings.TrimSpace(doc.Find("div.jILTFe").First().Text()); r != "" {
			if parsed, err := strconv.ParseFloat(strings.ReplaceAll(r, ",", ""), 64); err == nil {
				app.Rating = parsed
			}
		}
	}
	if app.RatingCount == 0 {
		// Try common selector
		if rc := strings.TrimSpace(doc.Find("span.EymY4b span").Last().Text()); rc != "" {
			rc = strings.ReplaceAll(rc, ",", "")
			rc = strings.Fields(rc)[0] // get number part
			if parsed, err := strconv.Atoi(rc); err == nil {
				app.RatingCount = parsed
			}
		}
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
