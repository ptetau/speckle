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
	case "commit":
		err = runCommit(os.Args[2:])
	case "new":
		err = runNew(os.Args[2:])
	case "validate":
		err = runValidate(os.Args[2:])
	case "log":
		err = runLog(os.Args[2:])
	case "show":
		err = runShow(os.Args[2:])
	case "inbox":
		err = runInbox(os.Args[2:])
	case "expand":
		err = runExpand(os.Args[2:])
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
  speckle serve    <file.speckle>        Render the plan as HTML and accept submissions.
  speckle await    <file.speckle>        Block until the running server receives a submit;
                                          print the submission JSON on stdout.
  speckle patch    <file.speckle>        Apply a YAML overlay from stdin to the file.
  speckle commit   <file.speckle>        Snapshot spec to git history repo.
                                          --decisions FILE  include decisions JSON as sidecar
                                          --message  MSG    commit message prefix (default: submit)
  speckle new      <file.speckle>        Write a commented starter spec to a new file.
  speckle validate <file.speckle>        Parse spec and print {"valid":true} or {"valid":false,...}.
  speckle log      <file.speckle>        List past decision rounds (newest first).
  speckle show     <file.speckle> <ref>  Print spec and decisions at a given git ref.
`)
}
