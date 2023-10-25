package main

import (
	"fmt"
	"os"

	"github.com/bensallen/nodeset"
	flag "github.com/spf13/pflag"
)

func main() {

	var pattern string

	flag.StringVarP(&pattern, "nodeset", "n", "", "nodeset pattern")
	flag.Parse()

	if pattern == "" {
		flag.Usage()
		os.Exit(1)
	}

	printer := func(s string) error { fmt.Println(s); return nil }
	err := nodeset.Expand(pattern, printer)
	if err != nil {
		fmt.Printf("Error expanding nodeset, %v.\n", err)
		os.Exit(1)
	}

}
