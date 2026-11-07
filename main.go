package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"hisab-sathi-api/internal/auth"
	"hisab-sathi-api/internal/database"
	"hisab-sathi-api/internal/datasync"
	"hisab-sathi-api/internal/landing"
)

func main() {
	// .env.local আসে `neon deploy` থেকে (DATABASE_URL), .env তে আমাদের নিজের সেটিংস।
	_ = godotenv.Load(".env.local", ".env")
	if os.Getenv("DATABASE_URL") == "" || os.Getenv("JWT_SECRET") == "" {
		log.Fatal("DATABASE_URL ও JWT_SECRET দরকার")
	}
	db, err := database.Open(context.Background())
	if err != nil {
		log.Fatalf("ডাটাবেস: %v", err)
	}
	defer db.Close()

	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.MaxMultipartMemory = 1 << 20
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:    []string{"Authorization", "Content-Type"},
	}))
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
	})

	landing.Register(r)
	r.GET("/health", func(c *gin.Context) {
		if err := db.Ping(c); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	auth.New(db).Register(r)
	datasync.New(db).Register(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(r.Run(":" + port))
}
