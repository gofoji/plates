package plates_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gofoji/plates"
)

// TestParseErrors verifies the Parse family reports parse failures eagerly, where From defers them to execution.
func TestParseErrors(t *testing.T) {
	f := plates.New("test").DefaultFunc(plates.TextParser).AddFuncs(plates.FuncMap{"echo": echo})

	_, err := f.Parse(`{{echo "ok"}}`)
	if err != nil {
		t.Errorf("Parse error = %v", err)
	}

	_, err = f.Parse("{{badFunc}}")
	if err == nil {
		t.Errorf("Parse: no error generated")
	}

	_, err = f.ParseName("x.txt.tpl", "{{badFunc}}")
	if err == nil {
		t.Errorf("ParseName: no error generated")
	}

	// From swallows the same failure and reports it at execution instead.
	_, err = f.From("{{badFunc}}").To(nil)
	if err == nil {
		t.Errorf("From: no error generated")
	}
}

func TestParseFile(t *testing.T) {
	f := plates.New("test").DefaultFunc(plates.TextParser).AddFuncs(plates.FuncMap{"echo": echo}).
		AddMatcherFunc(plates.MatchText, plates.MatchHTML, plates.MatchFormat)

	_, err := f.ParseFile("FILE DOES NOT EXIST")
	if err == nil {
		t.Errorf("ParseFile: no error generated")
	}

	f.FileReaderFunc(func(string) ([]byte, error) { return []byte(`{{echo "ok"}}`), nil })

	tpl, err := f.ParseFile("x.txt.tpl")
	if err != nil {
		t.Errorf("ParseFile error = %v", err)
		return
	}

	got, err := tpl.To(nil)
	if err != nil {
		t.Errorf("To error = %v", err)
		return
	}

	if want := "ECHO:'ok'"; got != want {
		t.Errorf("got = %v, want %v", got, want)
	}
}

// TestWithFuncs verifies a derived Factory shadows names without mutating the receiver.
func TestWithFuncs(t *testing.T) {
	base := plates.New("test").DefaultFunc(plates.TextParser).AddFuncs(plates.FuncMap{"echo": echo})

	derived := base.WithFuncs(plates.FuncMap{
		"echo":  func(string) string { return "SHOUT" },
		"extra": func() string { return "EXTRA" },
	})

	got, err := derived.From(`{{echo "x"}}{{extra}}`).To(nil)
	if err != nil {
		t.Errorf("derived error = %v", err)
		return
	}

	if want := "SHOUTEXTRA"; got != want {
		t.Errorf("derived got = %v, want %v", got, want)
	}

	got, err = base.From(`{{echo "x"}}`).To(nil)
	if err != nil {
		t.Errorf("base error = %v", err)
		return
	}

	want := "ECHO:'x'"
	if got != want {
		t.Errorf("base got = %v, want %v (receiver was mutated)", got, want)
	}

	_, err = base.Parse(`{{extra}}`)
	if err == nil {
		t.Errorf("base: derived func leaked into receiver")
	}
}

// TestWithFuncsMatcherIsolation verifies a derived Factory does not share the receiver's matcher backing array.
func TestWithFuncsMatcherIsolation(t *testing.T) {
	base := plates.New("test").DefaultFunc(plates.FormatParser).AddMatcherFunc(plates.MatchText)

	derived := base.WithFuncs(nil)
	derived.AddMatcherFunc(func(string) plates.Parser { return plates.ParserFunc(plates.HTMLParser) })

	// base still falls through to its default parser for an unmatched name.
	got, err := base.FromName("x.unmatched", "%s").To("raw<b>")
	if err != nil {
		t.Errorf("base error = %v", err)
		return
	}

	if want := "raw<b>"; got != want {
		t.Errorf("base got = %v, want %v (matchers were shared)", got, want)
	}
}

func TestScriptParser(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"trims and joins", "\n  echo \"a\"  \n\n\techo \"b\"\n", "ECHO:'a'ECHO:'b'"},
		{"drops whitespace only lines", "   \n\t\n", ""},
		{"empty", "", ""},
	}

	f := plates.New("test").DefaultFunc(plates.ScriptParser).AddFuncs(plates.FuncMap{"echo": echo})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.From(tt.in).To(nil)
			if err != nil {
				t.Errorf("error = %v", err)

				return
			}

			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}

	_, err := f.Parse("badFunc")
	if err == nil {
		t.Errorf("no error generated")
	}
}

func TestMatchScript(t *testing.T) {
	for _, name := range []string{"a.script", "a.goscript"} {
		if plates.MatchScript(name) == nil {
			t.Errorf("MatchScript(%q) = nil", name)
		}
	}

	for _, name := range []string{"a.txt.tpl", "a.gohtml", "a"} {
		if plates.MatchScript(name) != nil {
			t.Errorf("MatchScript(%q) != nil", name)
		}
	}
}

func TestToExecutor(t *testing.T) {
	f := plates.New("test").DefaultFunc(plates.TextParser).AddFuncs(plates.FuncMap{"echo": echo})

	tpl, err := f.Parse(`{{echo "ok"}}`)
	if err != nil {
		t.Errorf("Parse error = %v", err)
		return
	}

	var sb strings.Builder

	err = plates.ToExecutor(tpl).Execute(&sb, nil)
	if err != nil {
		t.Errorf("Execute error = %v", err)
		return
	}

	want := "ECHO:'ok'"
	if sb.String() != want {
		t.Errorf("got = %v, want %v", sb.String(), want)
	}

	// Errors propagate through the adapter.
	err = plates.ToExecutor(f.From("{{badFunc}}")).Execute(os.Stderr, nil)
	if err == nil {
		t.Errorf("no error generated")
	}
}
