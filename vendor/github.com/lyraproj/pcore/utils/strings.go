package utils

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

func AllStrings(strings []string, predicate func(str string) bool) bool {
	for _, v := range strings {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// ContainsString returns true if strings contains str
func ContainsString(strings []string, str string) bool {
	if str != `` {
		for _, v := range strings {
			if v == str {
				return true
			}
		}
	}
	return false
}

// ContainsAllStrings returns true if strings contains all entries in other
func ContainsAllStrings(strings []string, other []string) bool {
	for _, str := range other {
		if !ContainsString(strings, str) {
			return false
		}
	}
	return true
}

// IsDecimalInteger returns true if the string represents a base 10 integer
func IsDecimalInteger(s string) bool {
	if len(s) > 0 {
		for _, c := range s {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}

// MatchesString returns true if at least one of the regexps matches str
func MatchesString(regexps []*regexp.Regexp, str string) bool {
	if str != `` {
		for _, v := range regexps {
			if v.MatchString(str) {
				return true
			}
		}
	}
	return false
}

// MatchesAllStrings returns true if all strings are matched by at least one of the regexps
func MatchesAllStrings(regexps []*regexp.Regexp, strings []string) bool {
	for _, str := range strings {
		if !MatchesString(regexps, str) {
			return false
		}
	}
	return true
}

// Unique creates a new slice where all duplicate strings in the given slice have been removed. Order is retained
func Unique(strings []string) []string {
	top := len(strings)
	if top < 2 {
		return strings
	}
	exists := make(map[string]bool, top)
	result := make([]string, 0, top)

	for _, v := range strings {
		if !exists[v] {
			exists[v] = true
			result = append(result, v)
		}
	}
	return result
}

func CapitalizeSegment(segment string) string {
	b := bytes.NewBufferString(``)
	capitalizeSegment(b, segment)
	return b.String()
}

func capitalizeSegment(b *bytes.Buffer, segment string) {
	_, s := utf8.DecodeRuneInString(segment)
	if s > 0 {
		if s == len(segment) {
			b.WriteString(strings.ToUpper(segment))
		} else {
			b.WriteString(strings.ToUpper(segment[:s]))
			b.WriteString(strings.ToLower(segment[s:]))
		}
	}
}

var ColonSplit = regexp.MustCompile(`::`)

func CapitalizeSegments(segment string) string {
	segments := ColonSplit.Split(segment, -1)
	top := len(segments)
	if top > 0 {
		b := bytes.NewBufferString(``)
		capitalizeSegment(b, segments[0])
		for idx := 1; idx < top; idx++ {
			b.WriteString(`::`)
			capitalizeSegment(b, segments[idx])
		}
		return b.String()
	}
	return ``
}

func RegexpQuote(b io.Writer, str string) {
	WriteByte(b, '/')
	for _, c := range str {
		switch c {
		case '\t':
			WriteString(b, `\t`)
		case '\n':
			WriteString(b, `\n`)
		case '\r':
			WriteString(b, `\r`)
		case '/':
			WriteString(b, `\/`)
		case '\\':
			WriteString(b, `\\`)
		default:
			if c < 0x20 {
				_, err := fmt.Fprintf(b, `\u{%X}`, c)
				if err != nil {
					panic(err)
				}
			} else {
				WriteRune(b, c)
			}
		}
	}
	WriteByte(b, '/')
}

func PuppetQuote(w io.Writer, str string) {
	r := NewStringReader(str)
	b, ok := w.(*bytes.Buffer)
	if !ok {
		b = bytes.NewBufferString(``)
		defer func() {
			WriteString(w, b.String())
		}()
	}
	begin := b.Len()

	WriteByte(b, '\'')
	escaped := false
	for c := r.Next(); c != 0; c = r.Next() {
		if c < 0x20 {
			r.Rewind()
			b.Truncate(begin)
			puppetDoubleQuote(r, b)
			return
		}

		if escaped {
			WriteByte(b, '\\')
			WriteRune(b, c)
			escaped = false
			continue
		}

		switch c {
		case '\'':
			WriteString(b, `\'`)
		case '\\':
			escaped = true
		default:
			WriteRune(b, c)
		}
	}
	if escaped {
		WriteByte(b, '\\')
	}
	WriteByte(b, '\'')
}

func puppetDoubleQuote(r *StringReader, b io.Writer) {
	WriteByte(b, '"')
	for c := r.Next(); c != 0; c = r.Next() {
		switch c {
		case '\t':
			WriteString(b, `\t`)
		case '\n':
			WriteString(b, `\n`)
		case '\r':
			WriteString(b, `\r`)
		case '"':
			WriteString(b, `\"`)
		case '\\':
			WriteString(b, `\\`)
		case '$':
			WriteString(b, `\$`)
		default:
			if c < 0x20 {
				_, err := fmt.Fprintf(b, `\u{%X}`, c)
				if err != nil {
					panic(err)
				}
			} else {
				WriteRune(b, c)
			}
		}
	}
	WriteByte(b, '"')
}

func Fprintf(b io.Writer, format string, args ...interface{}) {
	_, err := fmt.Fprintf(b, format, args...)
	if err != nil {
		panic(err)
	}
}

func Fprintln(b io.Writer, args ...interface{}) {
	_, err := fmt.Fprintln(b, args...)
	if err != nil {
		panic(err)
	}
}

func WriteByte(b io.Writer, v byte) {
	_, err := b.Write([]byte{v})
	if err != nil {
		panic(err)
	}
}

func WriteRune(b io.Writer, v rune) {
	if v < utf8.RuneSelf {
		WriteByte(b, byte(v))
	} else {
		buf := make([]byte, utf8.UTFMax)
		n := utf8.EncodeRune(buf, v)
		_, err := b.Write(buf[:n])
		if err != nil {
			panic(err)
		}
	}
}

func WriteString(b io.Writer, s string) {
	_, err := io.WriteString(b, s)
	if err != nil {
		panic(err)
	}
}
