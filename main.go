// A simple file browser written in Go.
package main

import (
	"flag"
	"fmt"
	"log"
	"text/template"

	gitignore "github.com/sabhiram/go-gitignore"
)

var errNotFound = fmt.Errorf("not found")

var (
	bind           = flag.String("bind", ":8080", "address to bind to")
	dir            = flag.String("dir", "", "root directory to serve")
	out            = flag.String("out", "", "output directory for generated files")
	hideExtensions = flag.Bool("he", false, "hide file extensions")
	ignoreFile     = flag.String("ignore", ".ignore", "file with list of files to ignore")

	ignore        *gitignore.GitIgnore
	indexTemplate *template.Template

	connections = Connections{}
	contents    = Contents{}
)

func main() {
	flag.Parse()

	if *dir == "" {
		log.Fatal("dir is required")
	}

	runServer(bind)
}
