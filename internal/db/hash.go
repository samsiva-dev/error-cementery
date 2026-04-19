package db

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

var (
	reLineNumbers = regexp.MustCompile(`:\d+`)
	reWhitespace  = regexp.MustCompile(`\s+`)
	reHexAddr     = regexp.MustCompile(`0x[0-9a-fA-F]+`)
)

func NormalizeError(raw string) string {
	s := strings.ToLower(raw)
	s = reLineNumbers.ReplaceAllString(s, "")
	s = reHexAddr.ReplaceAllString(s, "0xaddr")
	s = reWhitespace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func HashError(raw string) string {
	h := sha256.Sum256([]byte(NormalizeError(raw)))
	return fmt.Sprintf("%x", h)
}
