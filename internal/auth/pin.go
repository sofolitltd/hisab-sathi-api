package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// verifyPin PIN মেলায়। ভুল হলে গণনা বাড়ে; ৫ বার ভুলে ১৫ মিনিট লক।
// ঠিক হলে ("", 0), নইলে (বার্তা, HTTP স্ট্যাটাস)।
func (h *Handler) verifyPin(c *gin.Context, id, hash string, failed int, lockedUntil *time.Time, pin string) (string, int) {
	if lockedUntil != nil && lockedUntil.After(time.Now()) {
		mins := int(time.Until(*lockedUntil).Minutes()) + 1
		return "অনেকবার ভুল PIN। " + strconv.Itoa(mins) + " মিনিট পরে চেষ্টা করুন", http.StatusTooManyRequests
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) != nil {
		failed++
		if failed >= maxAttempts {
			_, _ = h.db.Exec(c, `UPDATE users SET failed_attempts = 0, locked_until = $2 WHERE id = $1`,
				id, time.Now().Add(lockFor))
		} else {
			_, _ = h.db.Exec(c, `UPDATE users SET failed_attempts = $2 WHERE id = $1`, id, failed)
		}
		return "নম্বর বা PIN ভুল", http.StatusUnauthorized
	}
	_, _ = h.db.Exec(c, `UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = $1`, id)
	return "", 0
}
