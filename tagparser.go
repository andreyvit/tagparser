// Package tagparser is a better, simpler parser of conventional struct field
// tags, an alternative to vmihailenco/tagparser with a compact implementation,
// optional error reporting, and saner compatible tag syntax.
package tagparser

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

// ErrDuplicateKey is returned as Error.Cause for duplicate tag keys.
var ErrDuplicateKey = errors.New("duplicate option key")

// Error is the type of error returned by parse funcs in this package.
type Error struct {
	// Tag is the original tag string that has a syntax error.
	Tag string
	// Pos is a 0-based position within the Tag string appropriate to report
	// as errorneous.
	Pos int
	// Msg is an error message, or an optional prefix to the error message of
	// the Cause.
	Msg string
	// Cause is an optional underlying error returned by ParseFunc callback, or
	// ErrDuplicateKey.
	Cause error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		if e.Msg != "" {
			return fmt.Sprintf("%s: %v (at %d)", e.Msg, e.Cause, e.Pos+1)
		} else {
			return fmt.Sprintf("%v (at %d)", e.Cause, e.Pos+1)
		}
	} else {
		return fmt.Sprintf("%s (at %d)", e.Msg, e.Pos+1)
	}
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ParseName parses a tag treating the first item as a name. See ParseFunc for
// the full syntax and details.
func ParseName(tag string) (name string, opts map[string]string, err error) {
	err = ParseNameFunc(tag, func(key, value string) error {
		if key == "" {
			name = value
		} else {
			if opts == nil {
				opts = make(map[string]string)
			}
			if _, ok := opts[key]; ok {
				return ErrDuplicateKey
			}
			opts[key] = value
		}
		return nil
	})
	return
}

// Parse parses a tag without special treatment of the first item. See ParseFunc
// for the full syntax and details.
func Parse(tag string) (map[string]string, error) {
	var opts map[string]string
	err := ParseFunc(tag, func(key, value string) error {
		if opts == nil {
			opts = make(map[string]string)
		}
		if _, ok := opts[key]; ok {
			return ErrDuplicateKey
		}
		opts[key] = value
		return nil
	})
	return opts, err
}

// ParseNameFunc is like ParseFunc, but treats the first item as a name. See
// ParseFunc for the full syntax and details.
func ParseNameFunc(tag string, callback func(key, value string) error) error {
	return parseFunc(tag, true, callback)
}

// ParseFunc enumerates fields of a tag formatted as a list of keys and/or
// key-value pairs, optionally preceeded by a name.
//
// The format of the tag for ParseFunc and Parse is:
//
//	key1,key2:value2,key3:'quoted, value',key4
//
// The format of the tag for ParseNameFunc and ParseName is:
//
//	name,key1,key2:value2,key3:'quoted, value',key4
//
// Tag syntax:
//
//  1. A tag is a list of comma-separated items.
//
//  2. An item is either a key:value pair or just a single string.
//
//  3. Both keys and values can be bare words (`foo: bar`) or single-quoted
//     strings (`foo: 'bar: boz, buzz and fubar'`).
//
//  4. Both keys and values can use a backslash to escape special characters
//     (`foo\ bar`, `foo\:bar`, `foo\,bar`, `'foo\'n\'bar'`); the escapes are
//     processed and removed from the values (so `foo:\:\,\!` is returned as
//     `map[string]string{"foo": ":,!"}`); you can escape any non-alphabetical
//     characters;
//
//  5. Non-escaped unquoted leading and trailing ASCII whitespace is trimmed
//     from keys and values. (There seems to be no reason to handle Unicode
//     whitespace within struct tags.)
//
//  6. ParseName and ParseNameFunc give special treatment to the first item of
//     the tag if it does not have a colon. Such an item is returned as a name
//     output parameter by ParseName / as a value with an empty key by
//     ParseNameFunc. If the first item does have a colon, it is treated as a
//     normal key; ParseName returns an empty name, and Parse reports a normal
//     item and does not report an item with an empty key.
//
//  7. For normal items, empty key names are not allowed.
//
// The error, if present, is *Error. If your callback returns an error, it will
// be wrapped in an Error with your error stored in Error.Cause.
func ParseFunc(tag string, callback func(key, value string) error) error {
	return parseFunc(tag, false, callback)
}

func parseFunc(tag string, firstItemIsName bool, callback func(key, value string) error) error {
	var parseErr error
	fail := func(i int, msg string, cause error) {
		if parseErr == nil {
			parseErr = &Error{tag, i, msg, cause}
		}
	}

	var count int
	var inValue bool
	var start int
	var key string
	var keyStart int

	flush := func(i int) {
		count++
		var value, errMsg string
		var errPos int
		if count == 1 && firstItemIsName && !inValue {
			key = ""
			keyStart = start
			value, errMsg, errPos = unquoteTrim(tag[start:i])
			if errMsg != "" {
				fail(start+errPos, errMsg, nil)
			}
		} else {
			if inValue {
				key, errMsg, errPos = unquoteTrim(key)
				if errMsg != "" {
					fail(keyStart+errPos, errMsg, nil)
				}
				value, errMsg, errPos = unquoteTrim(tag[start:i])
				if errMsg != "" {
					fail(start+errPos, errMsg, nil)
				}
			} else if start < i {
				keyStart = start
				key, errMsg, errPos = unquoteTrim(tag[start:i])
				if errMsg != "" {
					fail(start+errPos, errMsg, nil)
				}
			} else {
				return
			}
			if key == "" {
				fail(keyStart, "empty key", nil)
				return
			}
		}
		err := callback(key, value)
		if err != nil {
			fail(keyStart, key, err)
		}
	}

	n := len(tag)

	checkEscape := func(i int) {
		if i >= n {
			fail(i-1, "unterminated escape sequence", nil)
			return
		}
		c := tag[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			fail(i, "invalid escape character", nil)
		}
	}

	var quoteStart int = -1
	for i := 0; i < n; i++ {
		if quoteStart >= 0 {
			switch tag[i] {
			case '\'':
				quoteStart = -1
			case '\\':
				i++
				checkEscape(i)
			}
		} else {
			switch tag[i] {
			case '\'':
				quoteStart = i
			case '\\':
				i++
				checkEscape(i)
			case ':':
				if !inValue {
					key = tag[start:i]
					keyStart = start
					start = i + 1
					inValue = true
				}
			case ',':
				flush(i)
				start = i + 1
				inValue = false
			}
		}
	}
	if quoteStart >= 0 {
		fail(quoteStart, "unterminated quote", nil)
	}
	if start < n || inValue {
		flush(n)
	}
	return parseErr
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// unquoteTrim trims leading and trailing unescaped ASCII whitespace, processes
// escape sequences within the string and removes single quotes.
func unquoteTrim(s string) (result string, parseErr string, errPos int) {
	n := len(s)

	var start int
	for start < n && asciiSpace[s[start]] != 0 {
		start++
	}

	var end int = n
	for end > start && asciiSpace[s[end-1]] != 0 {
		end--
	}
	// Note that end may have trimmed the final escaped space here. When we
	// encounter a backslash at s[end-1] and end < n, we will output s[end].

	if strings.IndexByte(s, '\\') < 0 && strings.IndexByte(s, '\'') < 0 {
		return s[start:end], "", 0
	}

	b := make([]byte, 0, n)
	var inQuote bool
	var quoteCount int
mainLoop:
	for i := start; i < end; i++ {
		c := s[i]
		switch c {
		case '\\':
			if i+1 < n {
				b = append(b, s[i+1])
				i++
			}
			continue mainLoop
		case '\'':
			quoteCount++
			if quoteCount > 2 || (quoteCount == 1 && len(b) > 0) {
				if parseErr == "" {
					parseErr, errPos = "invalid quote", i
				}
			}
			inQuote = !inQuote
			continue mainLoop
		}
		b = append(b, c)
	}
	if len(b) > 0 {
		result = unsafe.String(&b[0], len(b))
	}
	return
}
