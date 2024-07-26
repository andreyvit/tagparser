// Package tagparser is a better, simpler parser of conventional struct field
// tags, an alternative to the industry-standard vmihailenco/tagparser with a
// more compact implementation, optional error reporting, and an optionally 100%
// backwards compatible tag syntax.
package tagparser

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

type Configuration struct {
	FirstItemIsName  bool
	AllowParenEscape bool
	AllowMiddleQuote bool
}

var (
	WithName    = Configuration{FirstItemIsName: true}                                                 // WithName uses sane defaults and treats the first item as a name.
	WithoutName = Configuration{FirstItemIsName: false}                                                // WithoutName uses sane defaults and treats the first item like any other.
	VMihailenco = Configuration{FirstItemIsName: true, AllowParenEscape: true, AllowMiddleQuote: true} // VMihailenco matches the behavior of vmihailenco/tagparser.
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

// ParseName parses a tag formatted as a name followed by keys and/or
// key-value pairs. See ParseFunc for the full syntax.
func (conf *Configuration) ParseName(tag string) (name string, opts map[string]string, err error) {
	if !conf.FirstItemIsName {
		panic("tagparser: ParseName requires a configuration with FirstItemIsName = true")
	}
	err = conf.ParseFunc(tag, func(key, value string) error {
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

// ParseWithName parses a tag formatted as a list of keys and/or key-value
// pairs. See ParseFunc for the full syntax.
func (conf *Configuration) Parse(tag string) (map[string]string, error) {
	var opts map[string]string
	err := conf.ParseFunc(tag, func(key, value string) error {
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

// ParseFunc enumerates fields of a tag formatted as a list of keys and/or
// key-value pairs, optionally preceeded by a name.
//
// The format of the tag is: `name,key1,key2:value2,key3,key4=value4` when
// firstItemIsName is true, and a similar format without the name when
// firstItemIsName is false.
//
// When firstItemIsName is true, the name is reported as a value with an empty
// key. Note that if the first item is a key-value pair, no name will be
// reported.
//
// Keys and values can use single quotes to include special characters. In
// addition, anything inside parentheses, square brackets, or curly braces is
// treated as a single escaped value. You can use backslash escapes to escape
// whitespace, slashes, quotes and parentheses.
//
// Unescaped leading and trailing whitespace is trimmed from the keys and
// values. You can use quotes or escapes to add leading and trailing whitespace.
// Note that we're only processing ASCII whitespace, unlike strings.TrimSpace;
// there seems to be no reason to handle Unicode whitespace within struct tags.
//
// Empty key names are not allowed even if escaped with quotes. The empty key is
// reserved for the name when firstItemIsName is true.
//
// The error, if present, is *Error. If your callback returns an error, it will
// be wrapped in an Error with your error stored in Error.Cause.
func (conf *Configuration) ParseFunc(tag string, callback func(key, value string) error) error {
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
		if count == 1 && conf.FirstItemIsName && !inValue {
			key = ""
			keyStart = start
			value, errMsg, errPos = conf.unquoteTrim(tag[start:i])
			if errMsg != "" {
				fail(start+errPos, errMsg, nil)
			}
		} else {
			if inValue {
				key, errMsg, errPos = conf.unquoteTrim(key)
				if errMsg != "" {
					fail(keyStart+errPos, errMsg, nil)
				}
				value, errMsg, errPos = conf.unquoteTrim(tag[start:i])
				if errMsg != "" {
					fail(start+errPos, errMsg, nil)
				}
			} else if start < i {
				keyStart = start
				key, errMsg, errPos = conf.unquoteTrim(tag[start:i])
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
	var nesting int
	for i := 0; i < n; i++ {
		if quoteStart >= 0 {
			switch tag[i] {
			case '\'':
				quoteStart = -1
			case '\\':
				i++
				checkEscape(i)
			}
		} else if nesting > 0 {
			switch tag[i] {
			case ')', ']', '}':
				nesting--
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
			case '(', '[', '{':
				if conf.AllowParenEscape {
					nesting++
				}
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
func (conf *Configuration) unquoteTrim(s string) (result string, parseErr string, errPos int) {
	n := len(s)

	var start int
	for start < n && asciiSpace[s[start]] != 0 {
		start++
	}

	var end int = n
	for end > start && asciiSpace[s[end-1]] != 0 {
		end--
	}

	if strings.IndexByte(s, '\\') < 0 && strings.IndexByte(s, '\'') < 0 {
		return s[start:end], "", 0
	}

	b := make([]byte, 0, n)
	var inQuote bool
	var nesting int
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
		case '(', '[', '{':
			if conf.AllowParenEscape {
				nesting++
			}
		case ')', ']', '}':
			if conf.AllowParenEscape && nesting > 0 {
				nesting--
			}
		case '\'':
			if nesting == 0 {
				quoteCount++
				if !conf.AllowMiddleQuote {
					if quoteCount > 2 || (quoteCount == 1 && len(b) > 0) {
						if parseErr == "" {
							parseErr, errPos = "invalid quote", i
						}
					}
				}
				inQuote = !inQuote
				continue mainLoop
			}
		}
		b = append(b, c)
	}
	if len(b) > 0 {
		result = unsafe.String(&b[0], len(b))
	}
	return
}
