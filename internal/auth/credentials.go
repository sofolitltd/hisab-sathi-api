package auth

import (
	"regexp"
	"strings"
	"time"
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
