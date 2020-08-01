// +build ignore

// Bake is a helper for including static files in Go packages.
//
// Mostly scrounged from historic go.tools bake for which
// the following conditions apply:
//
// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"unicode/utf8"
)

var (
	out     = flag.String("o", "", "output file")
	pkgflag = flag.String("pkg", "", "package name to use")
)

func main() {
	flag.Parse()
	args := flag.Args()

	w := os.Stdout
	if *out != "" {
		if f, err := os.Create(*out); err == nil {
			w = f
			defer w.Close()
		} else {
			fatalln(err)
		}
	}

	pkg := *pkgflag
	if pkg == "" {
		pkg = os.Getenv("GOPACKAGE") // set when called from go generate
		if pkg == "" {
			fatalln("need package name")
		}
	}

	if err := bake(w, pkg, args); err != nil {
		fatalln(err)
	}
}

func bake(out io.Writer, pkg string, args []string) error {
	w := bufio.NewWriter(out)
	fmt.Fprintf(w, "%v\n\npackage %s\n\n", warning, pkg)

	for _, a := range args {
		name, file := split(a)
		fmt.Fprintf(w, "const %s = ", name)
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		if utf8.Valid(b) {
			fmt.Fprintf(w, "`%s`\n", sanitize(b))
		} else {
			fmt.Fprintf(w, "%q\n", b)
		}
	}

	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

func split(s string) (string, string) {
	ss := strings.SplitN(s, ":", 2)
	if len(ss) != 2 {
		fatalln("invalid argument:", s)
	}

	return ss[0], ss[1]
}

// sanitize prepares a valid UTF-8 string as a raw string constant.
func sanitize(b []byte) []byte {
	// Replace ` with `+"`"+`
	return bytes.Replace(b, []byte("`"), []byte("`+\"`\"+`"), -1)
}

func fatalln(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}

const warning = "// DO NOT EDIT ** This file was generated with the bake tool ** DO NOT EDIT //"
