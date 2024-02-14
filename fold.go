package nodeset

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type MatchGroup struct {
	Length            int                      // Length of original string
	NonDigitPositions map[int]string           // Key is position index with the non-digit component as a value
	DigitPadding      map[int]int              // Key is position index of digit elements, value is length of digit elements.
	DigitPositions    map[int]map[int]struct{} // Using a map of maps to only include unique digits: map[<pos. index>]map[<unique value>]struct{}
}

func Fold(inputs []string) []string {
	splitInput := make([][]string, len(inputs))
	for inputIndex, input := range inputs {
		splitInput[inputIndex] = splitOnDigits(input)
	}

	matchGroups := findMatchGroups(splitInput)
	output := make([]string, len(matchGroups))

	for groupIndex, group := range matchGroups {
		folded := ""
		for i := 0; i < group.Length; i++ {
			if _, ok := group.NonDigitPositions[i]; ok {
				folded += group.NonDigitPositions[i]
			} else if _, ok := group.DigitPositions[i]; ok {

				// Convert map[int]struct{} into a []int for numericRange()
				digitPositions := make([]int, len(group.DigitPositions[i]))
				j := 0
				for digit := range group.DigitPositions[i] {
					digitPositions[j] = digit
					j++
				}

				ranges, bracket := numericRange(digitPositions, group.DigitPadding[i])
				folded += formatRange(ranges, bracket)
			}
		}
		output[groupIndex] = folded
	}
	return output
}

// splitOnDigits splits an input string on any digits, where contigious charecters and digits are left together.
// "ab1000c" -> []string{"ab", "1000", "c"}
func splitOnDigits(s string) []string {
	var parts []string
	startChar := 0
	startDigit := 0
	foundChar := false
	foundDigit := false

	for i, char := range s {
		if unicode.IsDigit(char) {
			if !foundDigit {
				startDigit = i
				foundDigit = true
			}
			if foundChar {
				parts = append(parts, s[startChar:i])
				foundChar = false

			}
		} else {
			if !foundChar {
				startChar = i
				foundChar = true
			}
			if foundDigit {
				parts = append(parts, s[startDigit:i])
				foundDigit = false
			}
		}
	}
	//Add any trailing digits or charecters
	if foundDigit {
		parts = append(parts, s[startDigit:])
	} else if foundChar {
		parts = append(parts, s[startChar:])
	}
	return parts
}

func numericRange(input []int, padding int) ([]string, bool) {
	if len(input) == 0 {
		return []string{}, false
	}

	slices.Sort[[]int](input)

	var ranges []string
	var bracket bool

	start := input[0]
	end := input[0]

	for i := 1; i < len(input); i++ {
		if input[i] == end+1 {
			end = input[i]
		} else {
			if start == end {
				ranges = append(ranges, fmt.Sprintf("%0*d", padding, start))
			} else {
				ranges = append(ranges, fmt.Sprintf("%0*d-%0*d", padding, start, padding, end))
				bracket = true
			}
			start = input[i]
			end = input[i]
		}
	}

	if start == end {
		ranges = append(ranges, fmt.Sprintf("%0*d", padding, start))
	} else {
		ranges = append(ranges, fmt.Sprintf("%0*d-%0*d", padding, start, padding, end))
		bracket = true
	}

	if len(ranges) > 1 {
		bracket = true
	}

	return ranges, bracket
}

func formatRange(ranges []string, bracket bool) string {
	if bracket {
		return fmt.Sprintf("[%s]", strings.Join(ranges, ","))
	}
	return fmt.Sprintf("%s", strings.Join(ranges, ","))
}

func findMatchGroups(input [][]string) []MatchGroup {
	matchGroups := []MatchGroup{}

	positionMap := make(map[string]MatchGroup)

	for _, slice := range input {
		// Generate unique key using non-digit elements + length of digit elements
		// length is included as different
		key := ""
		for _, element := range slice {
			_, err := strconv.Atoi(element)
			if err != nil {
				key += element
			} else {
				key += strconv.Itoa(len(element))
			}
		}

		// Check if a group with the same key exists
		if group, ok := positionMap[key]; ok {
			// Group exists, append the slice to the existing group
			positionMap[key] = updatePositions(group, slice)
		} else {
			// Group doesn't exist, create a new group
			group := MatchGroup{
				Length:            len(slice),
				DigitPadding:      make(map[int]int),
				NonDigitPositions: make(map[int]string),
				DigitPositions:    make(map[int]map[int]struct{}),
			}
			positionMap[key] = updatePositions(group, slice)
		}
	}

	// Convert map values to slice
	for _, group := range positionMap {
		matchGroups = append(matchGroups, group)
	}

	return matchGroups
}

func updatePositions(group MatchGroup, slice []string) MatchGroup {
	for i, element := range slice {
		if val, err := strconv.Atoi(element); err == nil {
			if _, ok := group.DigitPositions[i]; !ok {
				group.DigitPositions[i] = make(map[int]struct{})
			}
			if _, ok := group.DigitPositions[i][val]; !ok {
				group.DigitPositions[i][val] = struct{}{}
				group.DigitPadding[i] = len(element)
			}
		} else {
			if group.NonDigitPositions[i] == "" {
				group.NonDigitPositions[i] = element
			}
		}
	}
	return group
}
