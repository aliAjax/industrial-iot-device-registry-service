package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func New(prefix string) string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(raw)
}

func NewWithTime(prefix string, now time.Time) string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return prefix + "_" + now.UTC().Format("20060102150405")
	}
	return prefix + "_" + now.UTC().Format("20060102150405") + "_" + hex.EncodeToString(raw)
}

func IsPrefixed(value, prefix string) bool {
	return strings.HasPrefix(value, prefix+"_")
}
