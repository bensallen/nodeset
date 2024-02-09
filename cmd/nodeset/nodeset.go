package main

import (
	"fmt"
	"os"

	"github.com/bensallen/nodeset"
	flag "github.com/spf13/pflag"
)

func main() {

	var patterns []string

	//flag.StringVarP(&pattern, "nodeset", "n", "", "nodeset pattern")
	flag.StringArrayVarP(&patterns, "nodeset", "n", []string{}, "nodeset pattern")
	flag.Parse()

	if len(patterns) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	for _, pattern := range patterns {
		for _, splitPattern := range nodeset.SplitOnComma(pattern) {
			printer := func(s string) error { fmt.Println(s); return nil }
			err := nodeset.Expand(splitPattern, printer)
			if err != nil {
				fmt.Printf("Error expanding nodeset, %v.\n", err)
				os.Exit(1)
			}
		}
	}
}
