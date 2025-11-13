package output

import (
	"fmt"
	"net/http"

	"github.com/dev-suryanshrajawat/Play-Store-Scrapping-Project/PlaystoreScrappingPro/parser"
	"github.com/gin-gonic/gin"
)

func ShowErrorPage(c *gin.Context, message string) {
	html := fmt.Sprintf(`<h2 style="color:red;">%s</h2><a href="/">⬅ Go Back</a>`, message)
	c.Data(200, "text/html", []byte(html))
}

func ShowAppInfo(c *gin.Context, app *parser.App) {
	html := fmt.Sprintf(`
<h2>Play Store App Info</h2>
<pre>
App Name/ID: %s
Developer: %s
Developer Email: %s
Developer Website: %s
Category: %s
Rating: %.1f
Total Ratings: %d
Installs: %s
Free: %t
Ad Supported: %t
In-App Purchases: %t
Last Updated: %s
Current Version: %s
Android Version: %s
Short Description: %s
Full Description: %s
</pre>
`, app.AppName, app.Developer, app.DeveloperEmail, app.DeveloperWebsite,
		app.Category, app.Rating, app.RatingCount, app.Installs, app.Free,
		app.AdSupported, app.InAppPurchase, app.LastUpdated,
		app.CurrentVersion, app.AndroidVersion, app.ShortDesc, app.Description)

	c.Data(200, "text/html", []byte(html))
}
