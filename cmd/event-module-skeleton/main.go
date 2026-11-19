package main

import (
	"flag"
	"fmt"
	"os"

	"tytd_go/internal/eventinspector"
)

func main() {
	name := flag.String("name", "", "snake_case module name")
	verified := flag.Bool("verified", false, "assert that an offline capture/fixture was reviewed")
	out := flag.String("out", "", "output Go file; stdout when empty")
	flag.Parse()
	s, err := eventinspector.GenerateModuleSkeleton(*name, *verified)
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
	if *out == "" { fmt.Print(s); return }
	if err := os.WriteFile(*out, []byte(s), 0600); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
