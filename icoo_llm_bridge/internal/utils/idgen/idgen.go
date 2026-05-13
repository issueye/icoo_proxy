package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

func New(prefix string) string {
	prefix = strings.Trim(strings.ToLower(prefix), "-_ ")
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		if prefix == "" {
			return time.Now().Format("20060102150405")
		}
		return prefix + "-" + time.Now().Format("20060102150405")
	}
	if prefix == "" {
		return hex.EncodeToString(data[:])
	}
	return prefix + "-" + hex.EncodeToString(data[:])
}
