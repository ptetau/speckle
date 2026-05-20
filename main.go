package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = runServe(os.Args[2:])
	case "await":
		err = runAwait(os.Args[2:])
	case "patch":
		err = runPatch(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "speckle:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `speckle — agent-driven spec building tool

Usage:
  speckle serve <file.speckle>   Render the plan as HTML and accept submissions.
  speckle await <file.speckle>   Block until the running server receives a submit;
                                  print the submission JSON on stdout.
  speckle patch <file.speckle>   Apply a YAML overlay from stdin to the file.
`)
}
