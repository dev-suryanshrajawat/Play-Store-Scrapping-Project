package main

import (
	"html/template"
	"net/http"

	"PLAYSTORESCRAPPER/output"
	"PLAYSTORESCRAPPER/parser"
	"PLAYSTORESCRAPPER/scraper"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.SetHTMLTemplate(template.Must(template.ParseFiles("templates/index.html")))

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/app-info", func(c *gin.Context) {
		pkg := c.Query("package")
		if pkg == "" {
			output.ShowErrorPage(c, "Package name is required")
			return
		}

		doc, err := scraper.FetchPlayStoreHTML(pkg)
		if err != nil {
			output.ShowErrorPage(c, err.Error())
			return
		}

		app, err := parser.ParsePlayStoreHTML(doc)
		if err != nil {
			output.ShowErrorPage(c, err.Error())
			return
		}

		output.ShowAppInfo(c, app)
	})

	r.Run(":8000")
}
