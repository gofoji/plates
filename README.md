# plates ![GitHub Workflow Status](https://img.shields.io/github/workflow/status/gofoji/plates/build) [![codecov](https://codecov.io/gh/gofoji/plates/branch/master/graph/badge.svg)](https://codecov.io/gh/gofoji/plates) [![PkgGoDev](https://pkg.go.dev/badge/github.com/gofoji/plates)](https://pkg.go.dev/github.com/gofoji/plates) [![Report card](https://goreportcard.com/badge/github.com/gofoji/plates)](https://goreportcard.com/report/github.com/gofoji/plates) [![Total alerts](https://img.shields.io/lgtm/alerts/g/gofoji/plates.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/gofoji/plates/alerts/) [![GitHub release](https://img.shields.io/github/release/gofoji/plates.svg?include_prereleases)](https://github.com/gofoji/plates/releases)

[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com)
[![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](https://forthebadge.com)

Plates is a simple abstraction to enable a simple [fluent](https://en.wikipedia.org/wiki/Fluent_interface) syntax for
template execution, and to allow runtime selection of the template engine.  Currently, Plates supports the go stdlib 
HTML and Text Parsers and the Buffalo Plush engine.  Note that the Plush support is in the subpackage 
`github.com/gofoji/plate/plush` to minimize dependencies.  If there are additional engines you would like to see, please 
reference the plush package to see how easy it is to add.

## Why

We needed a simple way for [foji](https://github.com/gofoji/foji) to support multiple template engines selected at runtime. 
After building this support we then needed it for a separate service and decided to abstract the package to let others
enjoy the functionality.

## Installation

```shell script
go get -u github.com/gofoji/plates
```

## Usage

### Simple Example

```go
out, err := plates.New("").
	DefaultFunc(plates.TextParser).
	From(`{{print "<h1>A header!</h1>"}}`).
	To(nil)
if err != nil {
    return err
}
```
Output
```html
<h1>A header!</h1>
```

### Advanced Example (from foji)

This shows an example of configuring the wrapper once and then using it to generate and execute multiple template instances.

```go
t := plates.New("").Funcs(runtime.Funcs, sprig.TxtFuncMap()).
	AddMatcherFunc(plates.MatchText, plush.Match).
    DefaultFunc(plates.MatchText)

targetFile, err := t.From(targetFile).To(data)
if err != nil {
    return err
}

err = t.FromFile(templateFile).ToFile(targetFile, data)
if err != nil {
    return err
}
```

The first call to `t.From()` uses the default parser (plates.MatchText).  The next parse call uses FromFile which uses the filename
to determine which template engine to invoke.  The selection is based on the order provided and if none of the filenames match it
falls back to the Default.

# TODO

- [ ] Identify additional template engines
- [ ] Review performance characteristics
- [ ] Godoc examples
