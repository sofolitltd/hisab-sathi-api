// Package datasync অ্যাপের লোকাল ডাটা ↔ সার্ভার সিঙ্ক (push / pull)।
//
// প্রতিটা রেকর্ডের uid সব ডিভাইসে এক; একই uid-তে নতুন updated_at জেতে।
// সার্ভারের seq দিয়ে ক্লায়েন্ট বোঝে কোথা থেকে আবার নামাতে হবে।
package datasync

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"hisab-sathi-api/internal/auth"
)

// Handler সিঙ্ক রুটগুলোর HTTP হ্যান্ডলার।
type Handler struct{ db *pgxpool.Pool }

// New ডাটাবেস পুল দিয়ে হ্যান্ডলার বানায়।
func New(db *pgxpool.Pool) *Handler { return &Handler{db: db} }

// Register `/sync/*` রুট বসায় (লগইন লাগে)।
func (h *Handler) Register(r *gin.Engine) {
	g := r.Group("/sync", auth.RequireAuth())
	g.POST("/push", h.push)
	g.GET("/pull", h.pull)
}
