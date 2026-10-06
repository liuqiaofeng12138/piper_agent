package util

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
