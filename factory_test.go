package plates_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofoji/plates"
)

func TestStringParsing(t *testing.T) {
	matchers := []plates.MatcherFunc{plates.MatchText, plates.MatchHTML, plates.MatchFormat}

	tests := []struct {
		name    string
		def     plates.ParserFunc
		mf      []plates.MatcherFunc
		f       []plates.FuncMap
		in      string
		data    interface{}
		want    string
		wantErr bool
	}{
		{"basic Text", plates.TextParser, matchers, []plates.FuncMap{{"echo": echo}}, `{{echo "<h1>A header!</h1>"}}`, nil, "ECHO:'<h1>A header!</h1>'", false},
		{"bad text", plates.TextParser, matchers, []plates.FuncMap{{"echo": echo}}, "{{badFunc}}", nil, "", true},
		{"basic HTML", plates.HTMLParser, matchers, []plates.FuncMap{{"echo": echo}}, `{{echo "<h1>A header!</h1>"}}`, nil, "ECHO:&#39;&lt;h1&gt;A header!&lt;/h1&gt;&#39;", false},
		{"bad HTML", plates.HTMLParser, matchers, []plates.FuncMap{{"echo": echo}}, `{{badFunc}}`, nil, "", true},
		{"abort", plates.TextParser, matchers, []plates.FuncMap{{"echo": echo}}, "{{.Abort }}", &TestContext{}, "", true},
		{"basic Format", plates.FormatParser, matchers, nil, `test%stest`, "foo", "testfootest", false},
		{"complex Format", plates.FormatParser, matchers, nil, `test%stest%s`, []interface{}{"1", "2"}, "test1test2", false},
		// Format does not allow errors for template parsing
		{"bad Format", plates.FormatParser, matchers, nil, `test%d%s%s%d%dtest`, "foo", "test%!d(string=foo)%!s(MISSING)%!s(MISSING)%!d(MISSING)%!d(MISSING)test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plates.New(tt.name).DefaultFunc(tt.def).AddFuncs(tt.f...).AddMatcherFunc(tt.mf...).From(tt.in).To(tt.data)
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
}

func TestFileParsing(t *testing.T) {
	matchers := []plates.MatcherFunc{plates.MatchText, plates.MatchHTML, plates.MatchFormat}
	funcs := plates.FuncMap{"echo": echo}
	def := plates.TextParser

	w := plates.New("test").DefaultFunc(def).AddFuncs(funcs).AddMatcherFunc(matchers...)

	ff, err := filepath.Glob("testdata/*")
	if err != nil {
		t.Errorf("glob error = %v", err)
	}

	for _, test := range ff {
		if strings.HasSuffix(test, "_want") {
			continue
		}

		t.Run(test, func(t *testing.T) {
			ctx := TestContext{}

			got, err := w.FromFile(test).To(&ctx)
			if err != nil {
				if strings.HasPrefix(test, "testdata/error") {
					return
				}
				t.Errorf("error = %v", err)
				return
			}

			want, err := os.ReadFile(test + "_want")
			if err != nil {
				t.Errorf("want error = %v", err)
				return
			}

			if got != string(want) {
				t.Errorf("got = `%v`, want `%v`", got, string(want))
			}
		})
	}

	w.FileReaderFunc(ErrorReader)

	_, err = w.FromFile("FILE DOES NOT EXIST").To(&TestContext{})
	if err == nil {
		t.Errorf("no error generated")
	}

	err = w.FromFile("FILE DOES NOT EXIST").ToFile("blah", &TestContext{})
	if err == nil {
		t.Errorf("no error generated")
	}

	err = w.FromFile("FILE DOES NOT EXIST").ToWriter(os.Stderr, &TestContext{})
	if err == nil {
		t.Errorf("no error generated")
	}

}

const tempOutputFile = "test_output.txt"

func TestEngine_ToFile(t *testing.T) {
	matchers := []plates.MatcherFunc{plates.MatchText, plates.MatchHTML}
	funcs := plates.FuncMap{"echo": echo}
	def := plates.TextParser

	e := plates.New("test").DefaultFunc(def).AddFuncs(funcs).AddMatcherFunc(matchers...).From(`{{echo "<h1>A header!</h1>"}}`)
	err := e.ToFile("stdout", &TestContext{})
	if err != nil {
		t.Errorf("stdout error = %v", err)
	}

	err = e.ToFile("stderr", &TestContext{})
	if err != nil {
		t.Errorf("stderr error = %v", err)
	}

	err = e.ToFile(tempOutputFile, &TestContext{})
	if err != nil {
		t.Errorf("testoutput error = %v", err)
	}

	e = plates.New("test").DefaultFunc(def).AddFuncs(funcs).AddMatcherFunc(matchers...).FileReaderFunc(ErrorReader).FromFile("blah")

	err = e.ToFile(tempOutputFile, &TestContext{})
	if err == nil {
		t.Errorf("testoutput error = %v", err)
	}
	_ = os.Remove(tempOutputFile)

	e = plates.New("test").DefaultFunc(def).AddFuncs(funcs).AddMatcherFunc(matchers...).FileReaderFunc(ErrorReader).From(`{{.Abort }}`)

	err = e.ToFile("does not matter", &TestContext{})
	if err == nil {
		t.Errorf("testoutput error = %v", err)
	}
}

func ErrorReader(name string) ([]byte, error) {
	return nil, fmt.Errorf("bad file: %s", name)
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

func (t *TestContext) String() string {
	return ""
}
