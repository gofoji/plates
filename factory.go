package plates

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
		To(data any) (string, error)
		ToFile(file string, data any) error
		ToWriter(w io.Writer, data any) error
	}

	// FuncMap maps template function names to functions.
	FuncMap map[string]any

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
		Execute(w io.Writer, data any) error
	}

	// ExecutorFunc executes a template against an io.Writer given the context.
	ExecutorFunc func(w io.Writer, data any) error
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

func (f ExecutorFunc) Execute(w io.Writer, data any) error {
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

// WithFuncs returns a derived Factory whose templates also see `in`.  Unlike AddFuncs it does not mutate the receiver,
// so a shared Factory can be specialized per consumer without the consumers observing each other's functions.
// Names in `in` shadow those already in the receiver.
func (t *Factory) WithFuncs(in ...FuncMap) *Factory {
	newT := *t
	funcs := make(FuncMap, len(t.funcs)+len(in))
	funcs.Add(t.funcs)
	funcs.Add(in...)

	newT.funcs = funcs
	newT.matchers = slices.Clone(t.matchers)

	return &newT
}

func (t *Factory) AddMatcher(in ...Matcher) *Factory {
	t.matchers = append(t.matchers, in...)

	return t
}

func (t *Factory) AddMatcherFunc(in ...MatcherFunc) *Factory {
	for _, mf := range in {
		t.matchers = append(t.matchers, mf)
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

func (t *Factory) FromFile(templateFile string) Template {
	return wrapErr(t.ParseFile(templateFile))
}

// FromName uses the engine matchers to find the appropriate template engine based on the name
func (t *Factory) FromName(name, s string) Template {
	return wrapErr(t.ParseName(name, s))
}

func (t *Factory) From(s string) Template {
	return wrapErr(t.Parse(s))
}

// ParseFile is the error returning form of FromFile.
func (t *Factory) ParseFile(templateFile string) (Template, error) {
	b, err := t.fileReader.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("error reading template: %s: %w", templateFile, err)
	}

	return t.ParseName(templateFile, string(b))
}

// ParseName is the error returning form of FromName.  Prefer it over FromName when the Template outlives the call
// that produced it, such as a long lived cache, so that a parse failure surfaces at load rather than at every render.
func (t *Factory) ParseName(name, s string) (Template, error) {
	for _, m := range t.matchers {
		parser := m.Match(name)
		if parser != nil {
			return t.parse(s, parser)
		}
	}

	return t.Parse(s)
}

// Parse is the error returning form of From.
func (t *Factory) Parse(s string) (Template, error) {
	return t.parse(s, t.defaultParser)
}

func (t *Factory) parse(s string, parser Parser) (Template, error) {
	e, err := parser.Parse(t.name, t.funcs, s)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	return e, nil
}

// wrapErr adapts the error returning Parse family to the fluent From family by deferring the error to execution.
func wrapErr(e Template, err error) Template {
	if err != nil {
		return errEngine{err}
	}

	return e
}

type engine struct {
	executor Executor
}

func NewEngine(executor Executor) Template {
	return &engine{executor: executor}
}

// ToExecutor adapts a Template to the Executor interface, the inverse of NewEngine.
func ToExecutor(t Template) Executor {
	return ExecutorFunc(func(w io.Writer, data any) error {
		return t.ToWriter(w, data)
	})
}

func (t *engine) ToWriter(w io.Writer, data any) error {
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

func (t *engine) ToBytes(data any) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := t.ToWriter(buf, data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *engine) To(data any) (string, error) {
	b, err := t.ToBytes(data)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (t *engine) ToFile(file string, data any) error {
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

	err = os.MkdirAll(filepath.Dir(file), 0o700)
	if err != nil {
		return fmt.Errorf("error creating output directory(`%s`): %w", filepath.Dir(file), err)
	}

	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
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

func (e errEngine) To(_ any) (string, error) {
	return "", e.err
}

func (e errEngine) ToFile(_ string, _ any) error {
	return e.err
}

func (e errEngine) ToWriter(_ io.Writer, _ any) error {
	return e.err
}

// Add takes the inbound FuncMaps and joins them to the current FuncMap.
func (ff FuncMap) Add(in ...FuncMap) {
	for _, funcs := range in {
		maps.Copy(ff, funcs)
	}
}
