package auth

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// userKey gin context-এ লগইন করা ইউজারের id রাখার চাবি।
const userKey = "uid"

// UserID [RequireAuth] পার হওয়া রিকুয়েস্টের ইউজার id।
func UserID(c *gin.Context) string { return c.GetString(userKey) }

func issueToken(userID string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(90 * 24 * time.Hour)),
	}).SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// RequireAuth Bearer টোকেন যাচাই করে user_id বসায় (পড়তে [UserID])।
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		claims := &jwt.RegisteredClaims{}
		tok, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !tok.Valid || claims.Subject == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "আবার লগইন করুন"})
			return
		}
		c.Set(userKey, claims.Subject)
		c.Next()
	}
}
