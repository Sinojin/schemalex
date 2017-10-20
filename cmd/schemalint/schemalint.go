package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/format"
)

var version = fmt.Sprintf("custom build (%s)", time.Now().Format(time.RFC3339))

func main() {
	if err := _main(); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
}

func _main() error {
	var showVersion bool
	var outfile string

	flag.Usage = func() {
		fmt.Printf(`schemalint version %s

schemalint -version
schemalint [options...] source

-v            Print out the version and exit
-o file	      Output the result to the specified file (default: stdout)

"source" may be a file path, or a URI.
Special URI schemes "mysql" and "local-git" are supported on top of
"file". If the special path "-" is used, it is treated as stdin.

Examples:

* Lint a local file
  schemalint /path/to/file
  schemalint file:///path/to/file

* Lint an online mysql schema
  schemalint "mysql://user:password@tcp(host:port)/dbname?option=value"

* Lint a file in local git repository 
  schemalint local-git:///path/to/repo?file=foo.sql&commitish=deadbeaf

* Lint schema from stdin against local file
	.... | schemalint -

`, version)
	}
	flag.BoolVar(&showVersion, "v", false, "")
	flag.StringVar(&outfile, "o", "", "")
	flag.Parse()

	if showVersion {
		fmt.Printf(
			"schemalint version %s, built with schemalex %s and go %s for %s/%s\n",
			schemalex.Version,
			version,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
		)
		return nil
	}

	if flag.NArg() != 1 {
		flag.Usage()
		return errors.New("wrong number of arguments")
	}

	var dst io.Writer = os.Stdout
	if len(outfile) > 0 {
		f, err := os.OpenFile(outfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return errors.Wrapf(err, `failed to open file %s for writing`, outfile)
		}
		dst = f
		defer f.Close()
	}

	fromSource, err := schemalex.NewSchemaSource(flag.Arg(0))
	if err != nil {
		return errors.Wrap(err, `failed to create schema source for "from"`)
	}

	var buf bytes.Buffer
	if err := fromSource.WriteSchema(&buf); err != nil {
		return errors.Wrap(err, `failed to read from source`)
	}

	p := schemalex.New()
	stmts, err := p.Parse(buf.Bytes())
	if err != nil {
		return errors.Wrap(err, `failed to parse source`)
	}

	for _, stmt := range stmts {
		if err := format.SQL(dst, stmt); err != nil {
			return errors.Wrap(err, `failed to format source`)
		}
		dst.Write([]byte{';', '\n', '\n'})
	}

	return nil
}