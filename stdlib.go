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

var ParserScript = ParserFunc(ScriptParser)

// ScriptParser parses a line oriented directive template.  Each line is the body of a single action *without*
// delimiters; blank lines are dropped and the rest are trimmed and wrapped in `{{- -}}`.  So:
//
//	$name := .Get "name"
//	.Set "greeting" (printf "hello %s" $name)
//
// becomes `{{- $name := .Get "name" -}}{{- .Set "greeting" (printf "hello %s" $name) -}}`.  The trim markers mean the
// template emits only what an action explicitly writes, and authors are free to indent for readability.  It is
// intended for templates that drive side effects rather than produce a document.
func ScriptParser(name string, funcs FuncMap, in string) (Template, error) {
	var final []string

	for line := range strings.SplitSeq(in, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			final = append(final, "{{- "+line+" -}}")
		}
	}

	return TextParser(name, funcs, strings.Join(final, "\n"))
}

// MatchScript returns a ScriptParser if the filename ends with ".script" or ".goscript"
func MatchScript(filename string) Parser {
	if strings.HasSuffix(filename, ".script") || strings.HasSuffix(filename, ".goscript") {
		return ParserFunc(ScriptParser)
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

func (f FormatEngine) Execute(w io.Writer, data any) error {
	var err error

	switch d := data.(type) {
	case []any:
		_, err = fmt.Fprintf(w, f.format, d...)
	default:
		_, err = fmt.Fprintf(w, f.format, d)
	}

	return err
}

var ParserFormat = ParserFunc(FormatParser)

func FormatParser(_ string, _ FuncMap, in string) (Template, error) {
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
