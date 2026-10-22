package shortcode

import (
	"errors"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var ErrInvalidCode = errors.New("invalid base62 code")

var decodeMap [256]int

func init() {
	for i := range decodeMap {
		decodeMap[i] = -1
	}
	for i := 0; i < len(alphabet); i++ {
		decodeMap[alphabet[i]] = i
	}
}

// Encode 把自增 ID 编码成短码。ID=1 -> "1",ID=62 -> "10"
func Encode(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [11]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = alphabet[n%62]
		n /= 62
	}
	return string(buf[i:])
}

// Decode 短码还原为 ID(本项目中跳转不依赖它,但统计/排查有用)
func Decode(s string) (uint64, error) {
	if s == "" {
		return 0, ErrInvalidCode
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		d := decodeMap[s[i]]
		if d < 0 {
			return 0, ErrInvalidCode
		}
		n = n*62 + uint64(d)
	}
	return n, nil
}
