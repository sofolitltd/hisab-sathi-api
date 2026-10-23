package main

import (
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	phoneRe = regexp.MustCompile(`^01[3-9][0-9]{8}$`)
	pinRe   = regexp.MustCompile(`^[0-9]{5}$`)
)

const (
	maxAttempts = 5
	lockFor     = 15 * time.Minute
)

type credentials struct {
	Phone string `json:"phone"`
	Pin   string `json:"pin"`
}

// normalizePhone +880 / 880 কেটে ০১XXXXXXXXX রূপে আনে।
func normalizePhone(p string) string {
	p = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, p)
	p = strings.TrimPrefix(p, "+88")
	p = strings.TrimPrefix(p, "88")
	return p
}

func (c *credentials) clean() string {
	c.Phone = normalizePhone(c.Phone)
	switch {
	case !phoneRe.MatchString(c.Phone):
		return "সঠিক মোবাইল নম্বর দিন (১১ ডিজিট)"
	case !pinRe.MatchString(c.Pin):
		return "PIN ঠিক ৫ ডিজিটের হতে হবে"
	}
	return ""
}

func issueToken(userID string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(90 * 24 * time.Hour)),
	}).SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func (s *server) register(c *gin.Context) {
	var in credentials
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ভুল অনুরোধ"})
		return
	}
	if msg := in.clean(); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Pin), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	var id string
	err = s.db.QueryRow(c, `INSERT INTO users (phone, pin_hash) VALUES ($1, $2) RETURNING id`,
		in.Phone, string(hash)).Scan(&id)
	if pe, ok := err.(*pgconn.PgError); ok && pe.Code == "23505" {
		c.JSON(http.StatusConflict, gin.H{"error": "এই নম্বরে আগেই অ্যাকাউন্ট আছে"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	s.respondToken(c, id, in.Phone)
}

// verifyPin PIN মেলায়। ভুল হলে গণনা বাড়ে; ৫ বার ভুলে ১৫ মিনিট লক।
// ঠিক হলে ("", 0), নইলে (বার্তা, HTTP স্ট্যাটাস)।
func (s *server) verifyPin(c *gin.Context, id, hash string, failed int, lockedUntil *time.Time, pin string) (string, int) {
	if lockedUntil != nil && lockedUntil.After(time.Now()) {
		mins := int(time.Until(*lockedUntil).Minutes()) + 1
		return "অনেকবার ভুল PIN। " + strconv.Itoa(mins) + " মিনিট পরে চেষ্টা করুন", http.StatusTooManyRequests
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) != nil {
		failed++
		if failed >= maxAttempts {
			_, _ = s.db.Exec(c, `UPDATE users SET failed_attempts = 0, locked_until = $2 WHERE id = $1`,
				id, time.Now().Add(lockFor))
		} else {
			_, _ = s.db.Exec(c, `UPDATE users SET failed_attempts = $2 WHERE id = $1`, id, failed)
		}
		return "নম্বর বা PIN ভুল", http.StatusUnauthorized
	}
	_, _ = s.db.Exec(c, `UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = $1`, id)
	return "", 0
}

func (s *server) login(c *gin.Context) {
	var in credentials
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ভুল অনুরোধ"})
		return
	}
	in.Phone = normalizePhone(in.Phone)
	var id, hash string
	var failed int
	var lockedUntil *time.Time
	err := s.db.QueryRow(c, `SELECT id, pin_hash, failed_attempts, locked_until FROM users WHERE phone = $1`,
		in.Phone).Scan(&id, &hash, &failed, &lockedUntil)
	if err != nil {
		// নম্বর আছে কি নেই তা বোঝানো হয় না; সময়ও সমান রাখা হয়।
		bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5BMFx7lLk1Qm6Z0tBcvFX0pQhBO5e"), []byte(in.Pin))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "নম্বর বা PIN ভুল"})
		return
	}
	if msg, code := s.verifyPin(c, id, hash, failed, lockedUntil, in.Pin); code != 0 {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	s.respondToken(c, id, in.Phone)
}

// changePin বর্তমান PIN দিয়ে নতুন ৫ ডিজিটের PIN বসায় (লগইন করা অবস্থায়)।
func (s *server) changePin(c *gin.Context) {
	var in struct {
		OldPin string `json:"old_pin"`
		NewPin string `json:"new_pin"`
	}
	if c.ShouldBindJSON(&in) != nil || !pinRe.MatchString(in.NewPin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "নতুন PIN ঠিক ৫ ডিজিটের হতে হবে"})
		return
	}
	id := c.GetString("uid")
	var hash string
	var failed int
	var lockedUntil *time.Time
	if err := s.db.QueryRow(c, `SELECT pin_hash, failed_attempts, locked_until FROM users WHERE id = $1`,
		id).Scan(&hash, &failed, &lockedUntil); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "আবার লগইন করুন"})
		return
	}
	if msg, code := s.verifyPin(c, id, hash, failed, lockedUntil, in.OldPin); code != 0 {
		if code == http.StatusUnauthorized {
			msg = "বর্তমান PIN ঠিক নয়"
		}
		c.JSON(code, gin.H{"error": msg})
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(in.NewPin), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	if _, err := s.db.Exec(c, `UPDATE users SET pin_hash = $2 WHERE id = $1`, id, string(newHash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *server) respondToken(c *gin.Context, id, phone string) {
	tok, err := issueToken(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "phone": phone})
}

// requireAuth Bearer টোকেন যাচাই করে user_id বসায়।
func requireAuth() gin.HandlerFunc {
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
		c.Set("uid", claims.Subject)
		c.Next()
	}
}
