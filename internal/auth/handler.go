// Package auth ফোন + ৫ ডিজিটের PIN দিয়ে অ্যাকাউন্ট খোলা, লগইন ও PIN বদল।
package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Handler auth রুটগুলোর HTTP হ্যান্ডলার।
type Handler struct{ db *pgxpool.Pool }

// New ডাটাবেস পুল দিয়ে হ্যান্ডলার বানায়।
func New(db *pgxpool.Pool) *Handler { return &Handler{db: db} }

// Register `/auth/*` রুট বসায়।
func (h *Handler) Register(r *gin.Engine) {
	r.POST("/auth/register", h.register)
	r.POST("/auth/login", h.login)
	r.POST("/auth/change-pin", RequireAuth(), h.changePin)
}

func (h *Handler) register(c *gin.Context) {
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
	err = h.db.QueryRow(c, `INSERT INTO users (phone, pin_hash) VALUES ($1, $2) RETURNING id`,
		in.Phone, string(hash)).Scan(&id)
	if pe, ok := err.(*pgconn.PgError); ok && pe.Code == "23505" {
		c.JSON(http.StatusConflict, gin.H{"error": "এই নম্বরে আগেই অ্যাকাউন্ট আছে"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	h.respondToken(c, id, in.Phone)
}

func (h *Handler) login(c *gin.Context) {
	var in credentials
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ভুল অনুরোধ"})
		return
	}
	in.Phone = normalizePhone(in.Phone)
	var id, hash string
	var failed int
	var lockedUntil *time.Time
	err := h.db.QueryRow(c, `SELECT id, pin_hash, failed_attempts, locked_until FROM users WHERE phone = $1`,
		in.Phone).Scan(&id, &hash, &failed, &lockedUntil)
	if err != nil {
		// নম্বর আছে কি নেই তা বোঝানো হয় না; সময়ও সমান রাখা হয়।
		bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5BMFx7lLk1Qm6Z0tBcvFX0pQhBO5e"), []byte(in.Pin))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "নম্বর বা PIN ভুল"})
		return
	}
	if msg, code := h.verifyPin(c, id, hash, failed, lockedUntil, in.Pin); code != 0 {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	h.respondToken(c, id, in.Phone)
}

// changePin বর্তমান PIN দিয়ে নতুন ৫ ডিজিটের PIN বসায় (লগইন করা অবস্থায়)।
func (h *Handler) changePin(c *gin.Context) {
	var in struct {
		OldPin string `json:"old_pin"`
		NewPin string `json:"new_pin"`
	}
	if c.ShouldBindJSON(&in) != nil || !pinRe.MatchString(in.NewPin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "নতুন PIN ঠিক ৫ ডিজিটের হতে হবে"})
		return
	}
	id := UserID(c)
	var hash string
	var failed int
	var lockedUntil *time.Time
	if err := h.db.QueryRow(c, `SELECT pin_hash, failed_attempts, locked_until FROM users WHERE id = $1`,
		id).Scan(&hash, &failed, &lockedUntil); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "আবার লগইন করুন"})
		return
	}
	if msg, code := h.verifyPin(c, id, hash, failed, lockedUntil, in.OldPin); code != 0 {
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
	if _, err := h.db.Exec(c, `UPDATE users SET pin_hash = $2 WHERE id = $1`, id, string(newHash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) respondToken(c *gin.Context, id, phone string) {
	tok, err := issueToken(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "phone": phone})
}
