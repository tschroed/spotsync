package spotsync

import (
	"strings"
	"unicode"
)

func CanonicalizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		// Contortions because Kanji, etc. are "other" and there's no tidy
		// test for "word characters".
		if unicode.IsControl(r) || unicode.IsMark(r) || unicode.IsPunct(r) || unicode.IsSpace(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
