package waotomatis_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// signFixture reproduces the server's HMAC-SHA256-over-raw-body signing so the
// webhook-verification tests use a known-good signature.
func signFixture(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
