package plush

import (
	"io"
	"strings"

	"github.com/gobuffalo/plush/v4"
	"github.com/gofoji/plates"
)

func Parser(name string, funcs plates.FuncMap, in string) (plates.Template, error) {
	t, err := plush.Parse(in)
	if err != nil {
		return nil, err
	}

	ctx := plush.NewContextWith(funcs)

	return plates.NewEngine(plushEngine{ctx, t}), nil
}

type plushEngine struct {
	ctx *plush.Context
	t   *plush.Template
}

func (e plushEngine) Execute(w io.Writer, data interface{}) error {
	ctx := e.ctx.New()
	ctx.Set("data", data)

	s, err := e.t.Exec(ctx)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(s))
	return err
}

// Match returns a Plush Parser if the filename ends with ".plush" or ".pls"
func Match(filename string) plates.Parser {
	if strings.HasSuffix(filename, ".plush") || strings.HasSuffix(filename, ".pls") {
		return plates.ParserFunc(Parser)
	}

	return nil
}
