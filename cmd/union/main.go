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
	switch os.Args[1] {
	case "build":
		fmt.Println("union build: not implemented")
	case "add":
		fmt.Println("union add: not implemented")
	case "explain":
		fmt.Println("union explain: not implemented")
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(`union - composable, versioned snippets for AGENTS.md / CLAUDE.md

usage:
  union build     compose snippets into a target file
  union add       add a snippet to the union
  union explain   show why each snippet is included`)
}
