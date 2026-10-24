package main

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// maxPhotoLen কম্প্রেস করা ছবির (base64 data URI) সর্বোচ্চ দৈর্ঘ্য।
const maxPhotoLen = 300 << 10

type customerDTO struct {
	UID    string `json:"uid"`
	Ledger string `json:"ledger"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	// পুরোনো অ্যাপ এটা পাঠায় না (nil) — তখন আগের মানই থাকে।
	ExtraPhones *string `json:"extra_phones"`
	// কিস্তির পরিকল্পনা (JSON লেখা, ফাঁকা = নেই); পুরোনো অ্যাপ পাঠায় না (nil)।
	Plan       *string `json:"plan"`
	Address    string  `json:"address"`
	Photo      string  `json:"photo"`
	DueDate    *int64  `json:"due_date"`
	ReminderAt *int64  `json:"reminder_at"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
	Deleted    bool    `json:"deleted"`
}

type entryDTO struct {
	UID         string  `json:"uid"`
	CustomerUID string  `json:"customer_uid"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Date        int64   `json:"date"`
	Note        string  `json:"note"`
	UpdatedAt   int64   `json:"updated_at"`
	Deleted     bool    `json:"deleted"`
}

// logDTO কাস্টমারকে কবে কল/মেসেজ/তাগাদা দেওয়া হয়েছিল।
type logDTO struct {
	UID         string `json:"uid"`
	CustomerUID string `json:"customer_uid"`
	Kind        string `json:"kind"`
	Note        string `json:"note"`
	At          int64  `json:"at"`
	UpdatedAt   int64  `json:"updated_at"`
	Deleted     bool   `json:"deleted"`
}

type pushBody struct {
	Customers []customerDTO `json:"customers"`
	Entries   []entryDTO    `json:"entries"`
	Logs      []logDTO      `json:"logs"`
}

// push: ক্লায়েন্টের নতুন/বদলানো রেকর্ড নেয়। একই uid-তে নতুন updated_at-ওয়ালা জেতে।
func (s *server) push(c *gin.Context) {
	var in pushBody
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ভুল অনুরোধ"})
		return
	}
	if len(in.Customers)+len(in.Entries)+len(in.Logs) > 5000 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "একবারে অনেক বেশি রেকর্ড"})
		return
	}
	user := c.GetString("uid")
	tx, err := s.db.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	defer tx.Rollback(c)

	for _, r := range in.Customers {
		if r.UID == "" || r.Name == "" || (r.Ledger != "pabo" && r.Ledger != "debo") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "কাস্টমারের তথ্য ভুল"})
			return
		}
		if len(r.Photo) > maxPhotoLen {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ছবি অনেক বড়"})
			return
		}
		_, err = tx.Exec(c, `
			INSERT INTO customers (user_id, uid, ledger, name, phone, address, photo, due_date, reminder_at, created_at, updated_at, deleted, extra_phones, plan)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,COALESCE($13,''),COALESCE($14,''))
			ON CONFLICT (user_id, uid) DO UPDATE SET
			  ledger = EXCLUDED.ledger, name = EXCLUDED.name, phone = EXCLUDED.phone,
			  extra_phones = COALESCE($13, customers.extra_phones),
			  plan = COALESCE($14, customers.plan),
			  address = EXCLUDED.address, photo = EXCLUDED.photo, due_date = EXCLUDED.due_date,
			  reminder_at = EXCLUDED.reminder_at, updated_at = EXCLUDED.updated_at,
			  deleted = EXCLUDED.deleted, seq = nextval('sync_seq')
			WHERE customers.updated_at <= EXCLUDED.updated_at`,
			user, r.UID, r.Ledger, r.Name, r.Phone, r.Address, r.Photo, r.DueDate, r.ReminderAt, r.CreatedAt, r.UpdatedAt, r.Deleted, r.ExtraPhones, r.Plan)
		if err != nil {
			log.Println("push:", err)
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23503" {
				// টোকেনের ইউজার ডাটাবেসে নেই (যেমন ডাটাবেস রিসেট) — আবার লগইন করাতে হবে।
				c.JSON(http.StatusUnauthorized, gin.H{"error": "আবার লগইন করুন"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
	}
	for _, r := range in.Entries {
		if r.UID == "" || r.CustomerUID == "" || r.Amount <= 0 || (r.Type != "baki" && r.Type != "joma") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "লেনদেনের তথ্য ভুল"})
			return
		}
		_, err = tx.Exec(c, `
			INSERT INTO entries (user_id, uid, customer_uid, type, amount, date, note, updated_at, deleted)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (user_id, uid) DO UPDATE SET
			  type = EXCLUDED.type, amount = EXCLUDED.amount, date = EXCLUDED.date,
			  note = EXCLUDED.note, updated_at = EXCLUDED.updated_at,
			  deleted = EXCLUDED.deleted, seq = nextval('sync_seq')
			WHERE entries.updated_at <= EXCLUDED.updated_at`,
			user, r.UID, r.CustomerUID, r.Type, r.Amount, r.Date, r.Note, r.UpdatedAt, r.Deleted)
		if err != nil {
			log.Println("push:", err)
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23503" {
				// টোকেনের ইউজার ডাটাবেসে নেই (যেমন ডাটাবেস রিসেট) — আবার লগইন করাতে হবে।
				c.JSON(http.StatusUnauthorized, gin.H{"error": "আবার লগইন করুন"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
	}
	for _, r := range in.Logs {
		switch r.Kind {
		case "call", "sms", "visit", "note":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "যোগাযোগের তথ্য ভুল"})
			return
		}
		if r.UID == "" || r.CustomerUID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "যোগাযোগের তথ্য ভুল"})
			return
		}
		_, err = tx.Exec(c, `
			INSERT INTO contact_logs (user_id, uid, customer_uid, kind, note, at, updated_at, deleted)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (user_id, uid) DO UPDATE SET
			  kind = EXCLUDED.kind, note = EXCLUDED.note, at = EXCLUDED.at,
			  updated_at = EXCLUDED.updated_at, deleted = EXCLUDED.deleted,
			  seq = nextval('sync_seq')
			WHERE contact_logs.updated_at <= EXCLUDED.updated_at`,
			user, r.UID, r.CustomerUID, r.Kind, r.Note, r.At, r.UpdatedAt, r.Deleted)
		if err != nil {
			log.Println("push:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
	}
	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// pull: `since` (সার্ভারের seq) এর পরের সব পরিবর্তন ফেরত দেয়।
func (s *server) pull(c *gin.Context) {
	since, _ := strconv.ParseInt(c.Query("since"), 10, 64)
	user := c.GetString("uid")
	cursor := since

	customers := []customerDTO{}
	rows, err := s.db.Query(c, `
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
	rows, err = s.db.Query(c, `
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
	rows, err = s.db.Query(c, `
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
