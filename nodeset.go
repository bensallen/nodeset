package nodeset

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Expand takes a node set pattern like 'node[1-2]', and a function
// with the signature func(s string). It will parse the pattern
// string and calculate the numerical ranges from the pattern.
// It will then create the Cartesian product for the pattern, for example:
// rack[1-2]node[3-4] ->
// rack1node3, rack1node4, rack2node3, rack2node4.
// Addition pattern syntax supported:
// Union ranges - node[1-2,5-9]
// Step ranges - node[1-4/2]
// The supplied iter function is called per Cartesian product.
func Expand(pattern string, iter func(s string) error) error {
	if pattern == "" {
		return fmt.Errorf("empty pattern")
	}
	if iter == nil {
		return fmt.Errorf("iter function nil")
	}
	ranges, err := splitInput(pattern)
	if err != nil {
		return err
	}

	// https://stackoverflow.com/a/29004530
	lens := func(i int) int { return len(ranges[i]) }

	for ix := make([]int, len(ranges)); ix[0] < lens(0); nextIndex(ix, lens) {
		var r []string
		for j, k := range ix {
			r = append(r, ranges[j][k])
		}
		err := iter(strings.Join(r, ""))
		if err != nil {
			return err
		}
	}
	return nil
}

// NextIndex sets ix to the lexicographically next value,
// such that for each i>0, 0 <= ix[i] < lens(i).
// https://stackoverflow.com/a/29004530
func nextIndex(ix []int, lens func(i int) int) {
	for j := len(ix) - 1; j >= 0; j-- {
		ix[j]++
		if j == 0 || ix[j] < lens(j) {
			return
		}
		ix[j] = 0
	}
}

func splitInput(input string) ([][]string, error) {
	var ranges [][]string

	for input != "" {
		if input[0] == '[' {
			end := 0
			for ; end < len(input) && input[end] != ']'; end++ {
				if end != 0 && input[end] == '[' {
					return [][]string{}, fmt.Errorf("input %s, contains a nested left bracket", input)
				}
			}
			if end == len(input) || input[end] != ']' {
				return [][]string{}, fmt.Errorf("input %s, contains a left bracket without a right bracket", input)
			}
			set, err := parseRange(input[:end+1])
			if err != nil {
				return [][]string{}, err
			}
			ranges = append(ranges, set)
			input = input[end+1:]
		} else {
			end := 0
			for ; end < len(input) && input[end] != '['; end++ {
				if input[end] == ']' {
					return [][]string{}, fmt.Errorf("input %s, contains a right bracket without a left bracket", input)
				}
			}

			ranges = append(ranges, []string{input[:end]})
			input = input[end:]
		}
	}
	return ranges, nil
}

// parseRange takes a string in the form of [1], [1-2], or [1-4/2]
// The returned range sets are deduplicated and numeric sorted.
func parseRange(rangeStr string) ([]string, error) {
	var rangeValues []string

	// Remove brackets from the range string
	if len(rangeStr) > 1 && rangeStr[0] == '[' && rangeStr[len(rangeStr)-1] == ']' {
		rangeStr = rangeStr[1 : len(rangeStr)-1]
	} else {
		return []string{}, fmt.Errorf("range [%s], is missing enclosing brackets", rangeStr)
	}

	// Split the range string by ','
	for _, index := range strings.Split(rangeStr, ",") {
		index, step, err := parseStep(index)
		if err != nil {
			return []string{}, err
		}

		rangeSplit := strings.Split(index, "-")

		if len(rangeSplit) == 1 {
			if step != 0 {
				return []string{}, fmt.Errorf("range [%s], contains a step without a start and stop range", index)
			}
			val, err := strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return []string{}, fmt.Errorf("range [%s], contains a single value that is not an integer", index)
			}
			rangeValues = append(rangeValues, strconv.FormatUint(val, 10))
		} else if len(rangeSplit) == 2 {
			start, err := strconv.ParseUint(rangeSplit[0], 10, 64)
			if err != nil {
				return []string{}, fmt.Errorf("range [%s], start with a value that is not an integer", index)
			}
			end, err := strconv.ParseUint(rangeSplit[1], 10, 64)
			if err != nil {
				return []string{}, fmt.Errorf("range [%s], ends with a value that is not an integer", index)
			}

			if start > end {
				return []string{}, fmt.Errorf("range [%s], starts with a value that is greater than the end value", index)
			}

			// If range start value has more than two characters and has a leading zero, assume that the output
			// should be padded to the same length as the start value.
			var padding int
			if len(rangeSplit[0]) > 1 && rangeSplit[0][0] == '0' {
				if len(rangeSplit[0]) > len(rangeSplit[1]) {
					return []string{}, fmt.Errorf("range [%s], zero padding on start value greater than end value length", index)
				}
				if rangeSplit[1][0] == '0' && (len(rangeSplit[0]) != len(rangeSplit[1])) {
					return []string{}, fmt.Errorf("range [%s], zero padding on end value must be same length as start value", index)
				}
				padding = len(rangeSplit[0])
			}

			// If step is its zero-value, default to incrementing by 1.
			if step == 0 {
				step = 1
			}

			for i := start; i <= end; i += step {
				rangeValues = append(rangeValues, fmt.Sprintf("%0*d", padding, i))
			}
		}
	}

	// Sort the values, safe to assume the strings are uint64 at this point
	slices.SortStableFunc(rangeValues, func(a, b string) int {
		an, _ := strconv.ParseUint(a, 10, 64)
		bn, _ := strconv.ParseUint(b, 10, 64)
		return cmp.Compare[uint64](an, bn)
	})
	return slices.Compact(rangeValues), nil
}

func parseStep(rangeStr string) (string, uint64, error) {
	var step uint64
	stepSplit := strings.Split(rangeStr, "/")
	if len(stepSplit) > 2 {
		return "", 0, fmt.Errorf("range [%s], contains more than one step delineator '/'", rangeStr)
	} else if len(stepSplit) == 2 {
		var err error
		step, err = strconv.ParseUint(stepSplit[1], 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("range [%s], contains a step that is not an integer", rangeStr)
		}
	}
	return stepSplit[0], step, nil
}
