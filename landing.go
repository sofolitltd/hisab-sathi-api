package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// landing/ এর HTML, CSS ও ছবি বাইনারির ভেতরেই থাকে।
//
//go:embed landing/index.html landing/app.css landing/logo.png landing/favicon.png landing/shots/*.jpg
var landingFS embed.FS

func registerLanding(r *gin.Engine) {
	sub, _ := fs.Sub(landingFS, "landing")
	page, _ := fs.ReadFile(sub, "index.html")

	r.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "text/html; charset=utf-8", page)
	})
	assets := http.StripPrefix("/assets/", http.FileServerFS(sub))
	r.GET("/assets/*file", func(c *gin.Context) {
		f := strings.TrimPrefix(c.Param("file"), "/")
		if f == "index.html" || f == "input.css" {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		assets.ServeHTTP(c.Writer, c.Request)
	})
}
