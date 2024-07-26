A better, simpler parser of conventional Go struct tags
=======================================================

[![Go reference](https://pkg.go.dev/badge/github.com/andreyvit/tagparser.svg)](https://pkg.go.dev/github.com/andreyvit/tagparser) ![Zero dependencies](https://img.shields.io/badge/deps-zero-brightgreen) ![Zero magic](https://img.shields.io/badge/magic-none-brightgreen) ![200 LOC](https://img.shields.io/badge/size-200%20LOC-green) ![100% coverage](https://img.shields.io/badge/coverage-100%25-green) [![Go Report Card](https://goreportcard.com/badge/github.com/andreyvit/tagparser)](https://goreportcard.com/report/github.com/andreyvit/tagparser)

Parses the conventional format of struct field tags: `name,key1,key2:value2,key3:'value with spaces: colons, commas, and \' quotes',key4`.

It's an alternative to [vmihailenco/tagparser](https://github.com/vmihailenco/tagparser) with a simpler implementation, optional error reporting, and saner compatible tag syntax.


Usage
-----

Install: `go get github.com/andreyvit/tagparser@latest`

Use `ParseName` for tags that treat the first item as a name:

```go
name, opts, err := tagparser.ParseName(`foo,bar,boz:'buzz fubar'`)
// name == "foo"
// opts == map[string]string{"bar": "", "boz": "buzz fubar"}
```

Use `Parse` for tags that don't need names:

```go
opts, err := tagparser.Parse(`foo,bar,boz:fubar`)
// opts == map[string]string{"foo": "", "bar": "", "boz": "fubar"}
```


Advanced usage
--------------

Use `ParseFunc` or `ParseNameFunc` for customized usage and even better error reporting (and zero allocations as a bonus):

```go
opts := make(map[string][]string)
callback := func(key, value string) error {
    if key != "foo" && key != "bar" {
        return errors.New("unsupported key")
    }
    opts[key] = append(opts[key], value)
    return nil
}

err := tagparser.ParseFunc(`foo,bar:xx,bar:yy`, callback)
// opts == map[string][]string{"foo": {""}, "bar": {"xx", "yy"}}

clear(opts)
err = tagparser.ParseNameFunc(`foo,bar:xx,bar:yy`, callback)
// opts == map[string][]string{"": {"foo"}, "bar": {"xx", "yy"}}

clear(opts)
err = tagparser.ParseFunc(`foo,boz,bar:xx`, callback)
// opts == map[string][]string{"foo": {""}, "boz": {""}, "bar": {"xx"}
// err.Error() = "boz: unsupported key (at 5)"
```


Error handling
--------------

All errors returned are `*tagparser.Error`, providing a reasonable message and a string index of the error. The content of the error is not covered by compatibility guarantees.

Note that you can simply ignore errors if you like; the parser still returns the best guess about the meaning of the tag.


Why?
----

This library is like [vmihailenco/tagparser](https://github.com/vmihailenco/tagparser), but:

* reports an error for incorrect tags (but also returns the best guess values, so you can ignore the error if you wish);
* gives a choice to treat the first item as name or not;
* has a consistent syntax without unexpected features;
* is a single ~200 LOC file — you can copy it into your project if you prefer not having a dependency;
* makes zero allocations when using `ParseFunc`, and only allocates the output map when using `ParseName` or `Parse`;
* has more tests and 100% test coverage.


Tag syntax
----------

* a tag is a list of comma-separated items;

* an item is either a `key:value` pair or just a single string;

* both keys and values can be bare words (`foo: bar`) or single-quoted strings (`foo: 'bar: boz, buzz and fubar'`);

* both keys and values can use a backslash to escape special characters (`foo\ bar`, `foo\:bar`, `foo\,bar`, `'foo\'n\'bar'`); the escapes are processed and removed from the values (so `foo:\:\,\!` is returned as `map[string]string{"foo": ":,!"}`); you can escape any non-alphabetical characters;

* non-escaped unquoted leading and trailing whitespace is trimmed from keys and values.

We are mostly compatible with vmihailenco/tagparser syntax, except we have:

* removed support for treating nested parenthesis as quotes: `foo: bar(boz, 'buzz', fubar)`, you should quote the entire value instead: `foo: 'bar(boz, \'buzz\', fubar)'`
* removed ability to use single quotes inside values: `foo: bar',buzz and 'fubar`, you should quote the entire value instead: `foo: 'bar,buzz and fubar'`


Contributing
------------

“We include what we think Go team would choose to include if this was a standard library package.”

We accept contributions that:

* add better documentation and examples;
* fix bugs;
* add extra error reporting;
* tweak tag syntax rules to be saner and more consistent, while staying compatible with vmihailenco/tagparser.

We recommend [modd](https://github.com/cortesi/modd) (`go install github.com/cortesi/modd/cmd/modd@latest`) for continuous testing during development.

Maintain 100% coverage. It's not often the right choice, but it is for this library.


BSD 2-Clause license
--------------------

Copyright © 2024, Andrey Tarantsov.

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
