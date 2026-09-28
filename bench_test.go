package nodeset

import (
	"strconv"
	"strings"
	"testing"
)

// genCray64k generates 65536 Cray-style hostnames of the form
// x<cabinet>c<chassis>s<slot>b<board>n<node> across a 16x8x8x8x8 topology.
func genCray64k() []string {
	nodes := make([]string, 0, 65536)
	var b strings.Builder
	for x := 0; x < 16; x++ {
		for c := 0; c < 8; c++ {
			for s := 0; s < 8; s++ {
				for bd := 0; bd < 8; bd++ {
					for n := 0; n < 8; n++ {
						b.Reset()
						b.WriteString("x")
						b.WriteString(strconv.Itoa(1000 + x))
						b.WriteString("c")
						b.WriteString(strconv.Itoa(c))
						b.WriteString("s")
						b.WriteString(strconv.Itoa(s))
						b.WriteString("b")
						b.WriteString(strconv.Itoa(bd))
						b.WriteString("n")
						b.WriteString(strconv.Itoa(n))
						nodes = append(nodes, b.String())
					}
				}
			}
		}
	}
	return nodes
}

func BenchmarkFoldCray64k(b *testing.B) {
	in := genCray64k()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Fold(in)
	}
}

// BenchmarkExpandCray64k expands the folded 5-dimensional Cray pattern back
// into its 65536 constituent hostnames.
func BenchmarkExpandCray64k(b *testing.B) {
	const pattern = "x[1000-1015]c[0-7]s[0-7]b[0-7]n[0-7]"
	noop := func(string) error { return nil }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Expand(pattern, noop); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkExpandFlat64k expands a single flat range of 65536 nodes.
func BenchmarkExpandFlat64k(b *testing.B) {
	const pattern = "node[0-65535]"
	noop := func(string) error { return nil }
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Expand(pattern, noop); err != nil {
			b.Fatal(err)
		}
	}
}
