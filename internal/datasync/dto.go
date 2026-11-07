package datasync

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
