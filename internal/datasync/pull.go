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

	products := []productDTO{}
	rows, err = h.db.Query(c, `
		SELECT uid, name, buy_price, sell_price, unit, updated_at, deleted, seq
		FROM products WHERE user_id = $1 AND seq > $2 ORDER BY seq`, user, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	for rows.Next() {
		var r productDTO
		var seq int64
		if err := rows.Scan(&r.UID, &r.Name, &r.BuyPrice, &r.SellPrice, &r.Unit, &r.UpdatedAt, &r.Deleted, &seq); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
		cursor = max(cursor, seq)
		products = append(products, r)
	}
	rows.Close()

	sales := []saleDTO{}
	rows, err = h.db.Query(c, `
		SELECT uid, product_uid, product_name, unit_buy_price, unit_sell_price, quantity,
		  customer_uid, received_amount, discount, date, note, profit, updated_at, deleted, seq
		FROM sales WHERE user_id = $1 AND seq > $2 ORDER BY seq`, user, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	for rows.Next() {
		var r saleDTO
		var seq int64
		if err := rows.Scan(&r.UID, &r.ProductUID, &r.ProductName, &r.UnitBuyPrice, &r.UnitSellPrice, &r.Quantity,
			&r.CustomerUID, &r.ReceivedAmount, &r.Discount, &r.Date, &r.Note, &r.Profit, &r.UpdatedAt, &r.Deleted, &seq); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
		cursor = max(cursor, seq)
		sales = append(sales, r)
	}
	rows.Close()

	purchases := []purchaseDTO{}
	rows, err = h.db.Query(c, `
		SELECT uid, product_uid, quantity, unit_cost, date, note, updated_at, deleted, seq
		FROM purchases WHERE user_id = $1 AND seq > $2 ORDER BY seq`, user, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	for rows.Next() {
		var r purchaseDTO
		var seq int64
		if err := rows.Scan(&r.UID, &r.ProductUID, &r.Quantity, &r.UnitCost, &r.Date, &r.Note,
			&r.UpdatedAt, &r.Deleted, &seq); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
		cursor = max(cursor, seq)
		purchases = append(purchases, r)
	}
	rows.Close()

	expenses := []expenseDTO{}
	rows, err = h.db.Query(c, `
		SELECT uid, category, amount, date, note, updated_at, deleted, seq
		FROM expenses WHERE user_id = $1 AND seq > $2 ORDER BY seq`, user, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
		return
	}
	for rows.Next() {
		var r expenseDTO
		var seq int64
		if err := rows.Scan(&r.UID, &r.Category, &r.Amount, &r.Date, &r.Note,
			&r.UpdatedAt, &r.Deleted, &seq); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "সার্ভারে সমস্যা"})
			return
		}
		cursor = max(cursor, seq)
		expenses = append(expenses, r)
	}
	rows.Close()

	c.JSON(http.StatusOK, gin.H{
		"customers": customers, "entries": entries, "logs": logs,
		"products": products, "sales": sales, "purchases": purchases,
		"expenses": expenses,
		"cursor":   cursor,
	})
}
