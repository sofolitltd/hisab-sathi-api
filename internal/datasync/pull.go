package datasync

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"hisab-sathi-api/internal/auth"
)

// pull: `since` (সার্ভারের seq) এর পরের সব পরিবর্তন ফেরত দেয়।
func (h *Handler) pull(c *gin.Context) {
	since, _ := strconv.ParseInt(c.Query("since"), 10, 64)
	user := auth.UserID(c)
	cursor := since

	customers := []customerDTO{}
	rows, err := h.db.Query(c, `
		SELECT uid, ledger, name, phone, extra_phones, plan, address, photo, due_date, reminder_at, created_at, updated_at, deleted, seq
		FROM customers WHERE user_id = $1 AND seq > $2 ORDER BY seq`, user, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	for rows.Next() {
		var r customerDTO
		var seq int64
		if err := rows.Scan(&r.UID, &r.Ledger, &r.Name, &r.Phone, &r.ExtraPhones, &r.Plan, &r.Address, &r.Photo, &r.DueDate, &r.ReminderAt,
			&r.CreatedAt, &r.UpdatedAt, &r.Deleted, &seq); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
		cursor = max(cursor, seq)
		customers = append(customers, r)
	}
	rows.Close()

	entries := []entryDTO{}
	rows, err = h.db.Query(c, `
		SELECT uid, customer_uid, type, amount, date, note, updated_at, deleted, seq
		FROM entries WHERE user_id = $1 AND seq > $2 ORDER BY seq`, user, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	for rows.Next() {
		var r entryDTO
		var seq int64
		if err := rows.Scan(&r.UID, &r.CustomerUID, &r.Type, &r.Amount, &r.Date, &r.Note,
			&r.UpdatedAt, &r.Deleted, &seq); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
		cursor = max(cursor, seq)
		entries = append(entries, r)
	}
	rows.Close()

	// অ্যাপ এই ঘর দেখেই বোঝে সার্ভার যোগাযোগের ইতিহাস রাখতে পারে
	logs := []logDTO{}
	rows, err = h.db.Query(c, `
		SELECT uid, customer_uid, kind, note, at, updated_at, deleted, seq
		FROM contact_logs WHERE user_id = $1 AND seq > $2 ORDER BY seq`, user, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	for rows.Next() {
		var r logDTO
		var seq int64
		if err := rows.Scan(&r.UID, &r.CustomerUID, &r.Kind, &r.Note, &r.At,
			&r.UpdatedAt, &r.Deleted, &seq); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
		cursor = max(cursor, seq)
		logs = append(logs, r)
	}
	rows.Close()

	c.JSON(http.StatusOK, gin.H{"customers": customers, "entries": entries, "logs": logs, "cursor": cursor})
}
