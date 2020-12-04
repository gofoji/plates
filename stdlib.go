package plates

import (
	htmlTemplate "html/template"
	"strings"
	"text/template"
)

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
	if strings.HasSuffix(filename, ".txt.tpl") || strings.HasSuffix(filename, ".gotxt")  || strings.HasSuffix(filename, ".tmpl") {
		return ParserFunc(TextParser)
	}

	return nil
}

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
	if strings.HasSuffix(filename, ".htm.tpl") || strings.HasSuffix(filename, ".html.tpl")  || strings.HasSuffix(filename, ".gohtml") {
		return ParserFunc(HTMLParser)
	}

	return nil
}
