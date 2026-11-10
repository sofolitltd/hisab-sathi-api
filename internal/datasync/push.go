package datasync

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"hisab-sathi-api/internal/auth"
)

// push: ক্লায়েন্টের নতুন/বদলানো রেকর্ড নেয়। একই uid-তে নতুন updated_at-ওয়ালা জেতে।
func (h *Handler) push(c *gin.Context) {
	var in pushBody
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ভুল অনুরোধ"})
		return
	}
	if len(in.Customers)+len(in.Entries)+len(in.Logs)+len(in.Products)+len(in.Sales)+len(in.Purchases)+len(in.Expenses) > 5000 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "একবারে অনেক বেশি রেকর্ড"})
		return
	}
	user := auth.UserID(c)
	tx, err := h.db.Begin(c)
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
		// due = "কবে দেবে" তারিখ বদল, অ্যাপ নিজে থেকে লেখে
		case "call", "sms", "visit", "note", "due":
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
	for _, r := range in.Products {
		if r.UID == "" || r.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "প্রোডাক্টের তথ্য ভুল"})
			return
		}
		_, err = tx.Exec(c, `
			INSERT INTO products (user_id, uid, name, buy_price, sell_price, unit, updated_at, deleted)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (user_id, uid) DO UPDATE SET
			  name = EXCLUDED.name, buy_price = EXCLUDED.buy_price, sell_price = EXCLUDED.sell_price,
			  unit = EXCLUDED.unit, updated_at = EXCLUDED.updated_at, deleted = EXCLUDED.deleted,
			  seq = nextval('sync_seq')
			WHERE products.updated_at <= EXCLUDED.updated_at`,
			user, r.UID, r.Name, r.BuyPrice, r.SellPrice, r.Unit, r.UpdatedAt, r.Deleted)
		if err != nil {
			log.Println("push:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
	}
	for _, r := range in.Sales {
		// ProductUID ফাঁকা থাকতে পারে — ইচ্ছাকৃতভাবে কোনো সেভড প্রোডাক্টের সাথে
		// link না থাকলে (শুধু ফ্রি-টেক্সট নাম)।
		if r.UID == "" || r.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "বিক্রয়ের তথ্য ভুল"})
			return
		}
		_, err = tx.Exec(c, `
			INSERT INTO sales (user_id, uid, product_uid, product_name, unit_buy_price, unit_sell_price,
			  quantity, customer_uid, received_amount, discount, date, note, profit, updated_at, deleted)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
			ON CONFLICT (user_id, uid) DO UPDATE SET
			  product_uid = EXCLUDED.product_uid, product_name = EXCLUDED.product_name,
			  unit_buy_price = EXCLUDED.unit_buy_price, unit_sell_price = EXCLUDED.unit_sell_price,
			  quantity = EXCLUDED.quantity, customer_uid = EXCLUDED.customer_uid,
			  received_amount = EXCLUDED.received_amount, discount = EXCLUDED.discount,
			  date = EXCLUDED.date, note = EXCLUDED.note,
			  profit = EXCLUDED.profit,
			  updated_at = EXCLUDED.updated_at, deleted = EXCLUDED.deleted, seq = nextval('sync_seq')
			WHERE sales.updated_at <= EXCLUDED.updated_at`,
			user, r.UID, r.ProductUID, r.ProductName, r.UnitBuyPrice, r.UnitSellPrice,
			r.Quantity, r.CustomerUID, r.ReceivedAmount, r.Discount, r.Date, r.Note, r.Profit, r.UpdatedAt, r.Deleted)
		if err != nil {
			log.Println("push:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
	}
	for _, r := range in.Purchases {
		if r.UID == "" || r.ProductUID == "" || r.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ক্রয়ের তথ্য ভুল"})
			return
		}
		_, err = tx.Exec(c, `
			INSERT INTO purchases (user_id, uid, product_uid, quantity, unit_cost, date, note, updated_at, deleted)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (user_id, uid) DO UPDATE SET
			  product_uid = EXCLUDED.product_uid, quantity = EXCLUDED.quantity, unit_cost = EXCLUDED.unit_cost,
			  date = EXCLUDED.date, note = EXCLUDED.note, updated_at = EXCLUDED.updated_at,
			  deleted = EXCLUDED.deleted, seq = nextval('sync_seq')
			WHERE purchases.updated_at <= EXCLUDED.updated_at`,
			user, r.UID, r.ProductUID, r.Quantity, r.UnitCost, r.Date, r.Note, r.UpdatedAt, r.Deleted)
		if err != nil {
			log.Println("push:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
	}
	for _, r := range in.Expenses {
		if r.UID == "" || r.Category == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ব্যয়ের তথ্য ভুল"})
			return
		}
		_, err = tx.Exec(c, `
			INSERT INTO expenses (user_id, uid, category, amount, date, note, updated_at, deleted)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (user_id, uid) DO UPDATE SET
			  category = EXCLUDED.category, amount = EXCLUDED.amount, date = EXCLUDED.date,
			  note = EXCLUDED.note, updated_at = EXCLUDED.updated_at,
			  deleted = EXCLUDED.deleted, seq = nextval('sync_seq')
			WHERE expenses.updated_at <= EXCLUDED.updated_at`,
			user, r.UID, r.Category, r.Amount, r.Date, r.Note, r.UpdatedAt, r.Deleted)
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
