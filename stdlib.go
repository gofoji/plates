package plates

import (
	"fmt"
	htmlTemplate "html/template"
	"io"
	"strings"
	"text/template"
)

var ParserText = ParserFunc(TextParser)

func TextParser(name string, funcs FuncMap, in string) (Template, error) {
	t := template.New(name)

	if len(funcs) > 0 {
		t.Funcs(template.FuncMap(funcs))
	}

	t, err := t.Parse(in)
	if err != nil {
		return nil, err
	}

	return NewEngine(t), nil
}

// MatchText returns a TextParser if the filename ends with ".txt.tpl"
func MatchText(filename string) Parser {
	if strings.HasSuffix(filename, ".txt.tpl") || strings.HasSuffix(filename, ".gotxt") || strings.HasSuffix(filename, ".tmpl") {
		return ParserFunc(TextParser)
	}

	return nil
}

var ParserHTML = ParserFunc(HTMLParser)

func HTMLParser(name string, funcs FuncMap, in string) (Template, error) {
	t := htmlTemplate.New(name)

	if len(funcs) > 0 {
		t.Funcs(htmlTemplate.FuncMap(funcs))
	}

	t, err := t.Parse(in)
	if err != nil {
		return nil, err
	}

	return NewEngine(t), nil
}

// MatchHTML returns an HTMLParser if the filename ends with ".htm.tpl" or ".html.tpl"
func MatchHTML(filename string) Parser {
	if strings.HasSuffix(filename, ".htm.tpl") || strings.HasSuffix(filename, ".html.tpl") || strings.HasSuffix(filename, ".gohtml") {
		return ParserFunc(HTMLParser)
	}

	return nil
}

// FormatEngine uses the golang fmt package to process a string.
// Warning: It is not possible to generate an error for an invalid format string.  Go will embed the error message in
// the output string, for example: `%!d(string=foo)` for an invalid type or `%!s(MISSING)` for missing data
type FormatEngine struct {
	format string
}

func (f FormatEngine) Execute(w io.Writer, data interface{}) error {
	var err error
	switch d := data.(type) {
	case []interface{}:
		_, err = fmt.Fprintf(w, f.format, d...)
	default:
		_, err = fmt.Fprintf(w, f.format, d)
	}
	return err
}

var ParserFormat = ParserFunc(FormatParser)

func FormatParser(name string, funcs FuncMap, in string) (Template, error) {
	t := FormatEngine{in}

	return NewEngine(t), nil
}

// MatchFormat returns a Go Format Parser if the filename ends with ".format" or ".string" or ".goformat"
func MatchFormat(filename string) Parser {
	if strings.HasSuffix(filename, ".format") || strings.HasSuffix(filename, ".string") || strings.HasSuffix(filename, ".goformat") {
		return ParserFunc(FormatParser)
	}

	return nil
}
