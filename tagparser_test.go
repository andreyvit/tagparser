package tagparser

import (
	"errors"
	"reflect"
	"testing"
)

type M = map[string]string

func TestParseWithName(t *testing.T) {
	var tests = []struct {
		testName string
		tag      string
		conf     Configuration
		name     string
		opts     map[string]string
		error    string
	}{
		{`empty`, ``, VMihailenco, "", nil, ``},

		{`simple 1`, `alfa`, VMihailenco, `alfa`, nil, ``},
		{`simple 2`, `alfa,bravo`, VMihailenco, `alfa`, M{"bravo": ""}, ``},

		{`quoted key 1`, `'alfa,bravo'`, VMihailenco, `alfa,bravo`, nil, ``},
		{`quoted key 2`, `'alfa:bravo'`, VMihailenco, `alfa:bravo`, nil, ``},
		{`quoted key 3`, `'alfa\:bravo'`, VMihailenco, `alfa:bravo`, nil, ``},
		{`quoted key 3`, `'alfa\:bravo'`, VMihailenco, `alfa:bravo`, nil, ``},
		{`quoted key 2`, "'alfa:bravo'", VMihailenco, `alfa:bravo`, nil, ``},

		{`escaped key 1`, `\ :alfa`, VMihailenco, "", M{" ": "alfa"}, ""},
		{`escaped key 1`, `' ':alfa`, VMihailenco, "", M{" ": "alfa"}, ""},

		{`no name 1`, `,alfa`, VMihailenco, "", M{"alfa": ""}, ``},
		{`no name 2`, `,alfa,bravo`, VMihailenco, "", M{"alfa": "", "bravo": ""}, ``},
		{`key with empty value`, `alfa:`, VMihailenco, "", M{"alfa": ""}, ``},
		{`key-value 1`, `alfa:bravo`, VMihailenco, "", M{"alfa": "bravo"}, ``},
		{`key-value 2`, `alfa:bravo,charlie`, VMihailenco, "", M{"alfa": "bravo", "charlie": ""}, ``},
		{`key-value 3`, `alfa:bravo,charlie:delta`, VMihailenco, "", M{"alfa": "bravo", "charlie": "delta"}, ``},

		{`whitespace 1`, `  alfa  `, VMihailenco, "alfa", nil, ``},
		{`whitespace 2`, ` alfa ,  bravo  `, VMihailenco, "alfa", M{"bravo": ""}, ``},
		{`whitespace 3`, ` alfa, charlie: delta `, VMihailenco, "alfa", M{"charlie": "delta"}, ``},

		{`skipped key`, `alfa,,charlie`, VMihailenco, "alfa", M{"charlie": ""}, ``},

		{`quoted value 1`, `alfa:'bravo,charlie'`, VMihailenco, "", M{"alfa": "bravo,charlie"}, ``},
		{`quoted value 2`, `alfa:'bravo,charlie',delta`, VMihailenco, "", M{"alfa": "bravo,charlie", "delta": ""}, ``},
		{`quoted value 3`, `alfa:'bravo:charlie',delta`, VMihailenco, "", M{"alfa": "bravo:charlie", "delta": ""}, ``},
		{`quoted value 4`, `alfa:'d\'Elta', bravo:charlie`, VMihailenco, "", M{"alfa": "d'Elta", "bravo": "charlie"}, ``},
		{`quote in the middle enabled`, `alfa:bravo', charlie 'delta`, VMihailenco, "", M{"alfa": "bravo, charlie delta"}, ``},
		{`disallowed quote in the middle 1`, `alfa:bravo', charlie 'delta`, WithName, "", M{"alfa": "bravo, charlie delta"}, `invalid quote (at 11)`},
		{`disallowed quote in the middle 2`, `alfa:'bravo 'charlie' delta'`, WithName, "", M{"alfa": "bravo charlie delta"}, `invalid quote (at 21)`},
		{`disallowed quote in the middle of name`, `bravo' charlie'`, WithName, "bravo charlie", nil, `invalid quote (at 6)`},
		{`disallowed quote in the middle of name`, `alfa,bravo' charlie'`, WithName, "alfa", M{"bravo charlie": ""}, `invalid quote (at 11)`},
		{`disallowed quote in the middle of key`, `bravo' charlie': delta`, WithName, "", M{"bravo charlie": "delta"}, `invalid quote (at 6)`},

		{`func value disabled`, `alfa:bravo('charlie', 'delta')`, VMihailenco, "", M{"alfa": "bravo('charlie', 'delta')"}, ``},

		{`func value with quotes`, `alfa:bravo('charlie', 'delta')`, VMihailenco, "", M{"alfa": "bravo('charlie', 'delta')"}, ``},
		{`func value nested`, `alfa:bravo('charlie', delta('boz'))`, VMihailenco, "", M{"alfa": "bravo('charlie', delta('boz'))"}, ``},
		{`func value with escapes`, `alfa:bravo{'charlie', \) delta 'boz'}`, VMihailenco, "", M{"alfa": "bravo{'charlie', ) delta 'boz'}"}, ``},
		{`func value with mismatched parens`, `alfa:bravo{'charlie'})))`, VMihailenco, "", M{"alfa": "bravo{'charlie'})))"}, ``},

		{`malformed empty key 1`, `alfa,:bravo`, VMihailenco, "alfa", nil, `empty key (at 6)`},
		{`malformed empty key 2`, `,:alfa`, VMihailenco, "", nil, `empty key (at 2)`},
		{`malformed empty key 3`, `'':alfa`, VMihailenco, "", nil, `empty key (at 1)`},
		{`malformed empty key 4`, ` '' :alfa`, VMihailenco, "", nil, `empty key (at 1)`},
		{`malformed duplicate key`, `alfa,bravo:charlie,bravo:delta`, VMihailenco, "alfa", M{"bravo": "charlie"}, `bravo: duplicate option key (at 20)`},
		{`malformed unterminated quote 1`, `alfa,'bravo:charlie`, VMihailenco, "alfa", M{"bravo:charlie": ""}, `unterminated quote (at 6)`},
		{`malformed unterminated quote 2`, `alfa,bravo:'charlie`, VMihailenco, "alfa", M{"bravo": "charlie"}, `unterminated quote (at 12)`},
		{`malformed unterminated quote 3`, `'alfa`, VMihailenco, "alfa", nil, `unterminated quote (at 1)`},
		{`malformed escape 1`, `a\lfa`, VMihailenco, "alfa", nil, `invalid escape character (at 3)`},
		{`malformed escape 2`, `al\`, VMihailenco, "al", nil, `unterminated escape sequence (at 3)`},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			name, opts, err := test.conf.ParseName(test.tag)
			if err != nil {
				ae := err.Error()
				if test.error == "" {
					t.Errorf("** Parse(%q) error %q, wanted no error", test.tag, ae)
				} else if ae != test.error {
					t.Errorf("** Parse(%q) error %q, wanted error %q", test.tag, ae, test.error)
				}
			} else if test.error != "" {
				t.Errorf("** Parse(%q) no error, wanted error %q", test.tag, test.error)
			}

			if name != test.name {
				t.Errorf("** Parse(%q) name = %q, wanted %q", test.tag, name, test.name)
			}

			if !reflect.DeepEqual(opts, test.opts) {
				for k, ev := range test.opts {
					av, ok := opts[k]
					if !ok {
						t.Errorf("** Parse(%q) missing option %q = %q", test.tag, k, ev)
					} else if av != ev {
						t.Fatalf("** Parse(%q) option %q = %q, wanted %q", test.tag, k, av, ev)
					}
				}
				for k, av := range opts {
					_, ok := test.opts[k]
					if !ok {
						t.Errorf("** Parse(%q) extra option %q = %q", test.tag, k, av)
					}
				}
			}
		})
	}
}

func TestParseWithoutName(t *testing.T) {
	var tests = []struct {
		testName string
		tag      string
		opts     map[string]string
		error    string
	}{
		{`empty`, ``, nil, ``},
		{`simple 1`, `alfa`, M{"alfa": ""}, ``},
		{`simple 2`, `alfa,bravo`, M{"alfa": "", "bravo": ""}, ``},
		{`key-value 1`, `alfa:bravo`, M{"alfa": "bravo"}, ``},
		{`key-value 2`, `alfa:bravo,charlie`, M{"alfa": "bravo", "charlie": ""}, ``},
		{`key-value 3`, `alfa:bravo,charlie:delta`, M{"alfa": "bravo", "charlie": "delta"}, ``},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			opts, err := WithoutName.Parse(test.tag)
			if err != nil {
				ae := err.Error()
				if test.error == "" {
					t.Errorf("** Parse(%q) error %q, wanted no error", test.tag, ae)
				} else if ae != test.error {
					t.Errorf("** Parse(%q) error %q, wanted error %q", test.tag, ae, test.error)
				}
			} else if test.error != "" {
				t.Errorf("** Parse(%q) no error, wanted error %q", test.tag, test.error)
			}

			if !reflect.DeepEqual(opts, test.opts) {
				for k, ev := range test.opts {
					av, ok := opts[k]
					if !ok {
						t.Errorf("** Parse(%q) missing option %q = %q", test.tag, k, ev)
					} else if av != ev {
						t.Fatalf("** Parse(%q) option %q = %q, wanted %q", test.tag, k, av, ev)
					}
				}
				for k, av := range opts {
					_, ok := test.opts[k]
					if !ok {
						t.Errorf("** Parse(%q) extra option %q = %q", test.tag, k, av)
					}
				}
			}
		})
	}
}

func TestParseWithoutName_duplicate(t *testing.T) {
	_, err := WithoutName.Parse(`foo:bar,foo:boz`)
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("** err = %v, wanted %v", err, ErrDuplicateKey)
	}
}

func TestParseWithName_panic(t *testing.T) {
	defer func() {
		e := recover()
		exp := "tagparser: ParseName requires a configuration with FirstItemIsName = true"
		if e != exp {
			t.Errorf("** panic = %v, wanted %q", e, exp)
		}
	}()
	_, _, _ = WithoutName.ParseName(`foo:bar,foo:boz`)
}

var errSimulated = errors.New("simulated error")

func TestParseWithFunc_custom_error_in_name(t *testing.T) {
	const tag = `foo,bar:boz`
	const expErr = `simulated error (at 1)`
	err := WithName.ParseFunc(tag, func(key, value string) error {
		if key == "" {
			return errSimulated
		}
		return nil
	})
	if err.Error() != expErr {
		t.Errorf("** error = %v, wanted %v", err, expErr)
	}
	if err.(*Error).Cause != errSimulated {
		t.Errorf("** Cause = %v, wanted %v", err, expErr)
	}
}

func TestParseWithFunc_custom_error_in_key(t *testing.T) {
	const tag = `foo,bar:boz`
	const expErr = `bar: simulated error (at 5)`
	err := WithName.ParseFunc(tag, func(key, value string) error {
		if key == "bar" {
			return errSimulated
		}
		return nil
	})
	if err.Error() != expErr {
		t.Errorf("** error = %v, wanted %v", err, expErr)
	}
	if err.(*Error).Cause != errSimulated {
		t.Errorf("** Cause = %v, wanted %v", err, expErr)
	}
}

func BenchmarkParseFunc(t *testing.B) {
	slice := make([]string, 0, 20)
	for i := 0; i < t.N; i++ {
		slice = slice[:0]
		err := WithName.ParseFunc(`foo,bar:boz,fubar,bar:zob,oof`, func(key, value string) error {
			slice = append(slice, key, value)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
