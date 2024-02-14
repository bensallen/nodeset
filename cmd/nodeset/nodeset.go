package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bensallen/nodeset"
	flag "github.com/spf13/pflag"
)

func main() {

	var expandNodeset bool
	var expandSeperator string
	var foldNodes bool
	var foldSeperator string

	//flag.StringVarP(&pattern, "nodeset", "n", "", "nodeset pattern")
	flag.BoolVarP(&expandNodeset, "expand", "e", false, "expand node sets to node list")
	flag.StringVarP(&expandSeperator, "expandSeperator", "S", " ", "deliminator for expanded node list")
	flag.BoolVarP(&foldNodes, "fold", "f", false, "fold node list into nodeset")
	flag.StringVarP(&foldSeperator, "foldSeperator", "s", ",", "deliminator for fold node list")

	flag.Parse()

	if !expandNodeset && !foldNodes {
		flag.Usage()
		os.Exit(1)
	}

	if expandNodeset && foldNodes {
		fmt.Println("Specifying expand and fold at the same time is unsupported.")
		flag.Usage()
		os.Exit(1)
	}

	// Attempt to interpret escape sequences, if any
	if interpreted, err := strconv.Unquote(`"` + expandSeperator + `"`); err == nil {
		expandSeperator = interpreted
	}
	if interpreted, err := strconv.Unquote(`"` + foldSeperator + `"`); err == nil {
		foldSeperator = interpreted
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		fmt.Println("Error checking if data is available via stdin")
	}
	stdinData := []byte{}
	if fi.Mode()&os.ModeNamedPipe != 0 {
		var err error
		stdinData, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("Error reading data via stdin")
		}
	}

	if expandNodeset {
		for _, set := range flag.Args() {
			for _, splitNodeset := range nodeset.SplitOnComma(set) {
				printer := func(s string) error { fmt.Printf("%s%s", s, expandSeperator); return nil }
				err := nodeset.Expand(splitNodeset, printer)
				if err != nil {
					fmt.Printf("Error expanding nodeset, %v.\n", err)
					os.Exit(1)
				}
			}
		}
	}

	if foldNodes {
		results := []string{}
		if len(stdinData) > 0 {
			inputs := strings.Fields(string(stdinData))
			results = nodeset.Fold(inputs)

		} else {
			results = nodeset.Fold(flag.Args())
		}
		fmt.Printf("%s\n", strings.Join(results, foldSeperator))
	}
}
