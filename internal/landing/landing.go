// Package landing ল্যান্ডিং পেজ, তার ছবি/CSS আর APK ডাউনলোডের লিংক দেয়।
package landing

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// web/ এর HTML, CSS ও ছবি বাইনারির ভেতরেই থাকে।
//
//go:embed web/index.html web/app.css web/logo.png web/favicon.png web/shots/*.jpg
var webFS embed.FS

const apkURL = "https://github.com/sofolitltd/hisab-sathi-app/releases/latest/download/hisab-sathi.apk"

// Register `/`, `/download` আর `/assets/*` রুট বসায়।
func Register(r *gin.Engine) {
	sub, _ := fs.Sub(webFS, "web")
	page, _ := fs.ReadFile(sub, "index.html")

	r.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "text/html; charset=utf-8", page)
	})
	// সর্বশেষ রিলিজের স্থির নামের APK-তে নেয়; নাম বদলায় না বলে ভার্সন বদলালেও লিংক ঠিক থাকে।
	r.GET("/download", func(c *gin.Context) {
		c.Redirect(http.StatusFound, apkURL)
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
