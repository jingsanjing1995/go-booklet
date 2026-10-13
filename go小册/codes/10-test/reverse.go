package reverse

import "unicode/utf8"

// ReverseRunes 按 Unicode 码点反转字符串。
func ReverseRunes(s string) string {
	rs := make([]rune, 0, utf8.RuneCountInString(s))
	for _, r := range s {
		rs = append(rs, r)
	}
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}
