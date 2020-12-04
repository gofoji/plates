package plush_test

import (
	"fmt"
	"testing"

	"github.com/gofoji/plate/plush"
	"github.com/gofoji/plates"
)

func TestStringParsing(t *testing.T) {
	tests := []struct {
		name    string
		def     plates.ParserFunc
		f       plates.FuncMap
		in      string
		data    interface{}
		want    string
		wantErr bool
	}{
		{"basic Plush", plush.Parser, plates.FuncMap{"echo": echo}, `<%= echo("<h1>A header!</h1>")%>`, nil, "ECHO:&#39;&lt;h1&gt;A header!&lt;/h1&gt;&#39;", false},
		{"bad Plush", plush.Parser, plates.FuncMap{"echo": echo}, `<%= INVALID SYNTAX %>`, nil, "", true},
		{"abort plush", plush.Parser, plates.FuncMap{"echo": echo}, "<% data.Abort() %>", &TestContext{}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plates.New(tt.name).DefaultFunc(tt.def).AddFuncs(tt.f).From(tt.in).To(tt.data)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("error = %v", err)
				}
				return
			}

			if tt.wantErr {
				t.Errorf("no error generated")
				return
			}

			if got != tt.want {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}

	matchers := []plates.MatcherFunc{plates.MatchText, plates.MatchHTML, plush.Match}
	funcs := plates.FuncMap{"echo": echo}
	def := plates.TextParser

	e := plates.New("test").FileReaderFunc(ConstReader).DefaultFunc(def).AddFuncs(funcs).AddMatcherFunc(matchers...).FromFile("whatever.plush")
	want := "ECHO:'TEST'"
	got, err := e.To(nil)
	if err != nil {
		t.Errorf("const error = %v", err)
	}

	if got != want {
		t.Errorf("got = %v, want %v", got, want)
	}

	e = plates.New("test").FileReaderFunc(ConstReader).DefaultFunc(def).AddFuncs(funcs).AddMatcherFunc(matchers...).FromFile("whatever.default")
	// Triggers default text parser which ignored the plush syntax
	want = "<%= raw(echo(\"TEST\"))%>"
	got, err = e.To(nil)
	if err != nil {
		t.Errorf("const error = %v", err)
	}

	if got != want {
		t.Errorf("got = %v, want %v", got, want)
	}

}

func ConstReader(_ string) ([]byte, error) {
	const t = `<%= raw(echo("TEST"))%>`
	return []byte(t), nil
}

func echo(in string) string {
	return fmt.Sprintf("ECHO:'%s'", in)
}

type TestContext struct {
	abort error
}

func (t *TestContext) Aborted() error {
	return t.abort
}

func (t *TestContext) Abort() string {
	t.abort = fmt.Errorf("aborted")
	return ""
}
