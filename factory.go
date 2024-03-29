package plates

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type (
	// FileReader allows overriding how filenames are evaluated and loaded.
	FileReader interface {
		ReadFile(filename string) ([]byte, error)
	}

	// FileReaderFunc allows overriding how filenames are evaluated and loaded.
	FileReaderFunc func(filename string) ([]byte, error)

	// Aborted is a special purpose error used to signal an aborted template execution.  It prevents saving output files.
	Aborted interface {
		Aborted() error
	}

	// Template is used to execute the parsed template and write the output to a string, File, or io.Writer.
	Template interface {
		To(data interface{}) (string, error)
		ToFile(file string, data interface{}) error
		ToWriter(w io.Writer, data interface{}) error
	}

	// FuncMap maps template function names to functions.
	FuncMap map[string]interface{}

	// Factory allows fluent execution of templates and runtime selection of the template engine.
	Factory struct {
		name          string
		fileReader    FileReader
		funcs         FuncMap
		defaultParser Parser
		matchers      []Matcher
	}

	// Matcher is used to match a file name to a Parser.
	Matcher interface {
		Match(filename string) Parser
	}

	// MatcherFunc is used to match a file name to a Parser.
	MatcherFunc func(string) Parser

	// Parser reads the input template string and returns an executable Template.
	Parser interface {
		Parse(name string, funcs FuncMap, input string) (Template, error)
	}

	// ParserFunc reads the input template string and returns an executable Template.
	ParserFunc func(name string, funcs FuncMap, input string) (Template, error)

	// Executor executes a template against an io.Writer given the context.
	Executor interface {
		Execute(w io.Writer, data interface{}) error
	}

	// ExecutorFunc executes a template against an io.Writer given the context.
	ExecutorFunc func(w io.Writer, data interface{}) error
)

func (f MatcherFunc) Match(filename string) Parser {
	return f(filename)
}

func (f ParserFunc) Parse(name string, funcs FuncMap, input string) (Template, error) {
	return f(name, funcs, input)
}

func (f FileReaderFunc) ReadFile(name string) ([]byte, error) {
	return f(name)
}

func (f ExecutorFunc) Execute(w io.Writer, data interface{}) error {
	return f(w, data)
}

// New creates a properly initialized Factory.  Name is an arbitrary label passed to the internal template engines
func New(name string) *Factory {
	return &Factory{name: name, fileReader: FileReaderFunc(os.ReadFile), funcs: FuncMap{}}
}

// FileReader allows overriding how filenames are evaluated and loaded.
func (t *Factory) FileReader(in FileReader) *Factory {
	t.fileReader = in

	return t
}

// FileReaderFunc allows overriding how filenames are evaluated and loaded.
func (t *Factory) FileReaderFunc(in FileReaderFunc) *Factory {
	return t.FileReader(in)
}

// AddFuncs adds the provided FuncMap to the library
func (t *Factory) AddFuncs(in ...FuncMap) *Factory {
	t.funcs.Add(in...)

	return t
}

func (t *Factory) AddMatcher(in ...Matcher) *Factory {
	t.matchers = append(t.matchers, in...)

	return t
}

func (t *Factory) AddMatcherFunc(in ...MatcherFunc) *Factory {
	for _, mf := range in {
		t.AddMatcher(mf)
	}

	return t
}

func (t *Factory) Default(in Parser) *Factory {
	t.defaultParser = in

	return t
}

func (t *Factory) DefaultFunc(in ParserFunc) *Factory {
	return t.Default(in)
}

func (t Factory) FromFile(templateFile string) Template {
	b, err := t.fileReader.ReadFile(templateFile)
	if err != nil {
		return errEngine{fmt.Errorf("error reading template: %s: %w", templateFile, err)}
	}

	return t.FromName(templateFile, string(b))
}

// FromName uses the engine matchers to find the approprite template engine based on the name
func (t Factory) FromName(name, s string) Template {
	for _, m := range t.matchers {
		parser := m.Match(name)
		if parser != nil {
			return t.from(s, parser)
		}
	}

	return t.From(s)
}

func (t Factory) From(s string) Template {
	return t.from(s, t.defaultParser)
}

func (t Factory) from(s string, parser Parser) Template {
	e, err := parser.Parse(t.name, t.funcs, s)
	if err != nil {
		return errEngine{fmt.Errorf("error parsing template: %w", err)}
	}

	return e
}

type engine struct {
	executor Executor
}

func NewEngine(executor Executor) Template {
	return &engine{executor: executor}
}

func (t *engine) ToWriter(w io.Writer, data interface{}) error {
	err := t.executor.Execute(w, data)

	a, ok := data.(Aborted)
	if ok {
		abortErr := a.Aborted()
		if abortErr != nil {
			return abortErr //nolint:wrapcheck
		}
	}

	return err //nolint:wrapcheck
}

func (t *engine) ToBytes(data interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := t.ToWriter(buf, data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *engine) To(data interface{}) (string, error) {
	b, err := t.ToBytes(data)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (t *engine) ToFile(file string, data interface{}) error {
	if file == "stdout" {
		return t.ToWriter(os.Stdout, data)
	}

	if file == "stderr" {
		return t.ToWriter(os.Stderr, data)
	}

	b, err := t.ToBytes(data)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return fmt.Errorf("error creating output directory(`%s`): %w", filepath.Dir(file), err)
	}

	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening file `%s`:%w", file, err)
	}

	_, err = f.Write(b)
	if closeErr := f.Close(); err != nil {
		return closeErr //nolint:wrapcheck
	}

	return err //nolint:wrapcheck
}

// errEngine is used to wrap a failed parse and still support fluent syntax
type errEngine struct{ err error }

func (e errEngine) To(_ interface{}) (string, error) {
	return "", e.err
}

func (e errEngine) ToFile(_ string, _ interface{}) error {
	return e.err
}

func (e errEngine) ToWriter(_ io.Writer, _ interface{}) error {
	return e.err
}

// Add takes the inbound FuncMaps and joins them to the current FuncMap.
func (ff FuncMap) Add(in ...FuncMap) {
	for _, funcs := range in {
		for name, fn := range funcs {
			ff[name] = fn
		}
	}
}
