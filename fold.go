package nodeset

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// token represents one component of a split node name. It is either a literal
// (non-digit) run of characters, or a digit value carrying its zero padding.
type token struct {
	isDigit bool
	literal string // set when isDigit is false
	value   uint64 // set when isDigit is true
	width   int    // original character width of the digit (for padding)
}

// node is a node name split into its constituent tokens.
type node []token

// Fold takes a list of node names and folds them into a compact set of
// bracketed range expressions.
//
// Folding proceeds from the rightmost digit dimension inward. Two nodes may
// only be merged on a given digit dimension when every other token (including
// any inner dimensions that were already folded) is identical. This prevents
// incorrectly producing a cartesian product across dimensions whose ranges do
// not match up. For example:
//
//	eh1f0, eh1f1, eh2f0, eh2f1, eh2f4
//
// folds to eh1f[0-1] and eh2f[0-1,4] rather than the incorrect
// eh[1-2]f[0-1,4].
func Fold(inputs []string) []string {
	// Deduplicate identical inputs while preserving the ability to fold.
	seen := make(map[string]struct{}, len(inputs))
	nodes := make([]node, 0, len(inputs))
	for _, input := range inputs {
		if _, ok := seen[input]; ok {
			continue
		}
		seen[input] = struct{}{}
		nodes = append(nodes, splitTokens(input))
	}

	// Repeatedly fold the rightmost unfolded digit dimension until no digit
	// dimensions remain that can be folded.
	nodes = foldAll(nodes)

	output := make([]string, len(nodes))
	for i, n := range nodes {
		output[i] = n.String()
	}
	return output
}

// foldAll folds every digit dimension, working from the rightmost dimension
// to the leftmost so that outer dimensions only merge when their inner
// (already folded) content is identical.
func foldAll(nodes []node) []node {
	// Determine the maximum number of digit dimensions across all nodes.
	maxDigits := 0
	for _, n := range nodes {
		count := 0
		for _, t := range n {
			if t.isDigit {
				count++
			}
		}
		if count > maxDigits {
			maxDigits = count
		}
	}

	// Fold from the innermost (rightmost) digit dimension outward.
	for dim := maxDigits - 1; dim >= 0; dim-- {
		nodes = foldDimension(nodes, dim)
	}
	return nodes
}

// foldDimension folds the digit dimension identified by index dim (0-based,
// counting digit tokens from the left). Nodes are grouped by their entire
// signature except for the value at dimension dim; each group's values at that
// dimension are collapsed into a single bracketed expression.
func foldDimension(nodes []node, dim int) []node {
	type group struct {
		representative node
		digitTokenIdx  int
		values         []token // the digit tokens at this dimension
	}

	groups := make(map[string]*group)
	var order []string
	var passthrough []node

	for _, n := range nodes {
		// Locate the token index of the dim-th digit token.
		digitTokenIdx := -1
		digitSeen := 0
		for i, t := range n {
			if t.isDigit {
				if digitSeen == dim {
					digitTokenIdx = i
					break
				}
				digitSeen++
			}
		}

		// If this node doesn't have a digit at this dimension, or it has
		// already been folded (represented as a literal range), pass it
		// through unchanged.
		if digitTokenIdx == -1 {
			passthrough = append(passthrough, n)
			continue
		}

		// Group by everything except the value at this dimension. All values
		// of the dimension share one bracket regardless of padding.
		key := signature(n, digitTokenIdx)
		g, ok := groups[key]
		if !ok {
			g = &group{
				representative: n,
				digitTokenIdx:  digitTokenIdx,
			}
			groups[key] = g
			order = append(order, key)
		}
		g.values = append(g.values, n[digitTokenIdx])
	}

	result := make([]node, 0, len(order)+len(passthrough))
	for _, key := range order {
		g := groups[key]

		folded := foldValues(g.values)

		// Replace the digit token with a literal containing the folded range.
		newNode := make(node, len(g.representative))
		copy(newNode, g.representative)
		newNode[g.digitTokenIdx] = token{isDigit: false, literal: folded}
		result = append(result, newNode)
	}

	result = append(result, passthrough...)
	return result
}

// foldValues collapses the digit tokens of a single dimension into a bracketed
// expression, matching ClusterShell semantics. Values are partitioned into
// pad-width classes: a padded class exists for each character width that a
// zero-padded value uses, and a value joins padded class W when its width
// equals W; all remaining values form an unpadded class rendered at natural
// width. Each class is range-collapsed independently. The unpadded class is
// emitted first, followed by padded classes in ascending width, all inside a
// single set of brackets.
func foldValues(values []token) string {
	// Determine which widths correspond to zero-padded (leading-zero) fields.
	padded := make(map[int]bool)
	for _, t := range values {
		if t.width > len(strconv.FormatUint(t.value, 10)) {
			padded[t.width] = true
		}
	}

	// Partition values by class: padded width, or 0 for the unpadded class.
	classes := make(map[int][]uint64)
	for _, t := range values {
		class := 0
		if padded[t.width] {
			class = t.width
		}
		classes[class] = append(classes[class], t.value)
	}

	// Emit unpadded class first (width 0), then padded classes ascending.
	widths := make([]int, 0, len(classes))
	for w := range classes {
		widths = append(widths, w)
	}
	slices.Sort(widths)

	var parts []string
	multi := false
	for _, w := range widths {
		ranges, bracket := numericRange(classes[w], w)
		if bracket {
			multi = true
		}
		parts = append(parts, ranges...)
	}

	if len(parts) > 1 {
		multi = true
	}
	return formatRange(parts, multi)
}

// signature builds a string uniquely identifying a node while ignoring the
// token at ignoreIdx. It is used to group nodes that differ only in the value
// at a single digit dimension.
func signature(n node, ignoreIdx int) string {
	var b strings.Builder
	var num [20]byte // scratch for uint64 formatting, avoids per-token allocs

	// Estimate size up front to grow the builder once rather than repeatedly.
	size := 0
	for i, t := range n {
		if i == ignoreIdx {
			size += 3
		} else if t.isDigit {
			size += 24 // marker + up to 20 digits + separators
		} else {
			size += len(t.literal) + 3
		}
	}
	b.Grow(size)

	for i, t := range n {
		if i == ignoreIdx {
			b.WriteString("\x00?\x00")
			continue
		}
		if t.isDigit {
			// Include the raw width so differently padded literals produced by
			// inner folds stay distinct.
			b.WriteString("\x00d")
			b.Write(strconv.AppendUint(num[:0], t.value, 10))
			b.WriteByte(':')
			b.Write(strconv.AppendUint(num[:0], uint64(t.width), 10))
			b.WriteByte(0)
		} else {
			b.WriteString("\x00l")
			b.WriteString(t.literal)
			b.WriteByte(0)
		}
	}
	return b.String()
}

// String renders a folded node back into its string form.
func (n node) String() string {
	var b strings.Builder
	var num [20]byte
	for _, t := range n {
		if t.isDigit {
			writePadded(&b, num[:], t.value, t.width)
		} else {
			b.WriteString(t.literal)
		}
	}
	return b.String()
}

// writePadded writes v to b, left-padded with zeros to at least width digits.
// scratch is a caller-provided buffer (>= 20 bytes) reused to avoid allocation.
func writePadded(b *strings.Builder, scratch []byte, v uint64, width int) {
	digits := strconv.AppendUint(scratch[:0], v, 10)
	for pad := width - len(digits); pad > 0; pad-- {
		b.WriteByte('0')
	}
	b.Write(digits)
}

// splitTokens splits a node name into literal and digit tokens.
// "ab1000c" -> [literal "ab", digit 1000 (width 4), literal "c"]
//
// Digit runs are parsed directly here, avoiding the redundant ParseUint calls
// (and swallowed errors) that classifying via string parts would incur. A run
// of ASCII digits that overflows uint64 is retained as a literal token.
func splitTokens(s string) node {
	tokens := make(node, 0, tokenCountHint(s))

	start := 0
	// inDigit tracks the classification of the current run; runStarted guards
	// the first character.
	var inDigit, runStarted bool

	flush := func(end int) {
		part := s[start:end]
		if inDigit {
			if v, err := strconv.ParseUint(part, 10, 64); err == nil {
				tokens = append(tokens, token{isDigit: true, value: v, width: len(part)})
				return
			}
			// Overflowed uint64; keep the digits as a literal.
		}
		tokens = append(tokens, token{literal: part})
	}

	for i := 0; i < len(s); i++ {
		d := s[i] >= '0' && s[i] <= '9'
		if !runStarted {
			runStarted = true
			inDigit = d
			start = i
			continue
		}
		if d != inDigit {
			flush(i)
			inDigit = d
			start = i
		}
	}
	if runStarted {
		flush(len(s))
	}
	return tokens
}

// tokenCountHint estimates the number of tokens in s by counting transitions
// between digit and non-digit runs, so splitTokens can preallocate.
func tokenCountHint(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	prevDigit := s[0] >= '0' && s[0] <= '9'
	for i := 1; i < len(s); i++ {
		d := s[i] >= '0' && s[i] <= '9'
		if d != prevDigit {
			count++
			prevDigit = d
		}
	}
	return count
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

func numericRange(input []uint64, padding int) ([]string, bool) {
	if len(input) == 0 {
		return []string{}, false
	}

	slices.Sort(input)
	input = slices.Compact(input)

	var ranges []string
	var bracket bool
	var b strings.Builder
	var num [20]byte

	emit := func(start, end uint64) {
		b.Reset()
		writePadded(&b, num[:], start, padding)
		if start != end {
			b.WriteByte('-')
			writePadded(&b, num[:], end, padding)
			bracket = true
		}
		ranges = append(ranges, b.String())
	}

	start := input[0]
	end := input[0]

	for i := 1; i < len(input); i++ {
		// input is sorted and deduplicated, so input[i] > end holds here;
		// subtract rather than add to avoid overflow at math.MaxUint64.
		if input[i]-end == 1 {
			end = input[i]
		} else {
			emit(start, end)
			start = input[i]
			end = input[i]
		}
	}
	emit(start, end)

	if len(ranges) > 1 {
		bracket = true
	}

	return ranges, bracket
}

func formatRange(ranges []string, bracket bool) string {
	joined := strings.Join(ranges, ",")
	if bracket {
		return "[" + joined + "]"
	}
	return joined
}
