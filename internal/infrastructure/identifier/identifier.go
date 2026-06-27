package identifier

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	nonAlpha    = regexp.MustCompile(`[^a-z0-9]+`)
	multiHyphen = regexp.MustCompile(`-{2,}`)
)

func Make(s string) string {
	s = strings.ToLower(s)
	s = nonAlpha.ReplaceAllString(s, "-")
	s = multiHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

// Unique returns an identifier that does not exist according to the provided predicate.
// If the base identifier is taken, numeric suffixes are appended (-2, -3, ...).
func Unique(base string, exists func(string) bool) string {
	candidate := base
	for i := 2; ; i++ {
		if !exists(candidate) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// GenerateCode derives a 4-char uppercase code from a name.
// Takes the first 4 uppercase alphanumeric characters; pads with 'X' if needed.
func GenerateCode(name string) string {
	var out []rune
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, unicode.ToUpper(r))
			if len(out) == 4 {
				break
			}
		}
	}
	for len(out) < 4 {
		out = append(out, 'X')
	}
	return string(out)
}

// UniqueCode returns a 4-char code that does not exist according to the predicate.
// If the base code is taken, the last character(s) are replaced with a numeric suffix.
func UniqueCode(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}
	for i := 2; ; i++ {
		suffix := strconv.Itoa(i)
		n := 4 - len(suffix)
		if n < 0 {
			n = 0
		}
		candidate := base[:n] + suffix
		if !exists(candidate) {
			return candidate
		}
	}
}
