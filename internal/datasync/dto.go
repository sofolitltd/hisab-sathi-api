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

// productDTO ক্রয়-বিক্রয়ের প্রোডাক্ট (সাধারণ পণ্য বা মোবাইল ব্যাংকিং/কার্ড কমিশন আইটেম)।
type productDTO struct {
	UID       string  `json:"uid"`
	Name      string  `json:"name"`
	BuyPrice  float64 `json:"buy_price"`
	SellPrice float64 `json:"sell_price"`
	Unit      string  `json:"unit"`
	UpdatedAt int64   `json:"updated_at"`
	Deleted   bool    `json:"deleted"`
}

// saleDTO এক বিক্রয়ের রেকর্ড — দামের ঘরগুলো বিক্রয়ের সময়ের স্ন্যাপশট।
type saleDTO struct {
	UID            string  `json:"uid"`
	ProductUID     string  `json:"product_uid"`
	ProductName    string  `json:"product_name"`
	UnitBuyPrice   float64 `json:"unit_buy_price"`
	UnitSellPrice  float64 `json:"unit_sell_price"`
	Quantity       float64 `json:"quantity"`
	CustomerUID    string  `json:"customer_uid"`
	ReceivedAmount float64 `json:"received_amount"`
	// ইচ্ছাকৃত ছাড় — "বাকি" থেকে আলাদা, কাস্টমারের ledger-এ যোগ হয় না।
	Discount float64 `json:"discount"`
	Date     int64   `json:"date"`
	Note     string  `json:"note"`
	// nil = লাভ অজানা (প্রোডাক্ট লিংক নেই আর হাতে লেখাও হয়নি)।
	Profit    *float64 `json:"profit"`
	UpdatedAt int64    `json:"updated_at"`
	Deleted   bool     `json:"deleted"`
}

// purchaseDTO একটা রিস্টক/ক্রয়ের রেকর্ড।
type purchaseDTO struct {
	UID        string  `json:"uid"`
	ProductUID string  `json:"product_uid"`
	Quantity   float64 `json:"quantity"`
	UnitCost   float64 `json:"unit_cost"`
	Date       int64   `json:"date"`
	Note       string  `json:"note"`
	UpdatedAt  int64   `json:"updated_at"`
	Deleted    bool    `json:"deleted"`
}

// expenseDTO ক্যাটেগরি-ভিত্তিক সাধারণ ব্যয় (দোকান ভাড়া, খাবার, বাজার ইত্যাদি)।
type expenseDTO struct {
	UID       string  `json:"uid"`
	Category  string  `json:"category"`
	Amount    float64 `json:"amount"`
	Date      int64   `json:"date"`
	Note      string  `json:"note"`
	UpdatedAt int64   `json:"updated_at"`
	Deleted   bool    `json:"deleted"`
}

type pushBody struct {
	Customers []customerDTO `json:"customers"`
	Entries   []entryDTO    `json:"entries"`
	Logs      []logDTO      `json:"logs"`
	Products  []productDTO  `json:"products"`
	Sales     []saleDTO     `json:"sales"`
	Purchases []purchaseDTO `json:"purchases"`
	Expenses  []expenseDTO  `json:"expenses"`
}
