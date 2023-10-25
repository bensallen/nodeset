package nodeset

import (
	"fmt"
	"reflect"
	"testing"
)

func TestExpand(t *testing.T) {
	type args struct {
		pattern string
		iter    func(s string) error
	}

	var output []string
	funcArg := func(s string) error {
		output = append(output, s)
		return nil
	}
	funcErrArg := func(s string) error {
		return fmt.Errorf("this is an error")
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{

		{
			name: "No range",
			args: args{pattern: "node1", iter: funcArg},
			want: []string{"node1"},
		},
		{
			name: "Range value",
			args: args{pattern: "node[1-2]", iter: funcArg},
			want: []string{"node1", "node2"},
		},
		{
			name:    "Empty pattern",
			args:    args{pattern: "", iter: funcArg},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "No iter",
			args:    args{pattern: "node1", iter: nil},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Nested left bracket, error passed up from splitInput",
			args:    args{pattern: "node[[1]", iter: funcArg},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Iter with error",
			args:    args{pattern: "node[1-2]", iter: funcErrArg},
			want:    []string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		output = []string{}
		t.Run(tt.name, func(t *testing.T) {
			err := Expand(tt.args.pattern, tt.args.iter)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(output, tt.want) {
				t.Errorf("Expand() = %v, want %v", output, tt.want)
			}
		})
	}
}

func Test_splitInput(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		want    [][]string
		wantErr bool
	}{
		{
			name: "No range",
			args: args{input: "node1"},
			want: [][]string{{"node1"}},
		},
		{
			name: "No step, range value",
			args: args{input: "node[1-2]"},
			want: [][]string{{"node"}, {"1", "2"}},
		},
		{
			name: "Overlapping union ranges",
			args: args{input: "node[1-2,1-4]"},
			want: [][]string{{"node"}, {"1", "2", "3", "4"}},
		},
		{
			name: "Step, range value",
			args: args{input: "node[1-4/2]"},
			want: [][]string{{"node"}, {"1", "3"}},
		},
		{
			name: "Multiple range values",
			args: args{input: "x100[1-2]c[3-4]"},
			want: [][]string{{"x100"}, {"1", "2"}, {"c"}, {"3", "4"}},
		},
		{
			name:    "Left bracket but no right bracket",
			args:    args{input: "node[1"},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name:    "Nested left bracket",
			args:    args{input: "node[[1]"},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name:    "Right bracket but no left bracket",
			args:    args{input: "x1001]c0"},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name:    "Left bracket but no right bracket, multiple ranges",
			args:    args{input: "x1001]c[0-1]"},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name:    "Nested right bracket",
			args:    args{input: "node[1]]"},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name:    "Nested brackets",
			args:    args{input: "node[[1]]"},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name:    "Single value range non-integer, passing error up from parseRange",
			args:    args{input: "node[a]"},
			want:    [][]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitInput(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseRange(t *testing.T) {
	type args struct {
		rangeStr string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "No step, single value",
			args: args{rangeStr: "[1]"},
			want: []string{"1"},
		},
		{
			name: "No step, range value",
			args: args{rangeStr: "[1-2]"},
			want: []string{"1", "2"},
		},
		{
			name: "Step, range value",
			args: args{rangeStr: "[1-4/2]"},
			want: []string{"1", "3"},
		},
		{
			name: "Step, range value with zero padding",
			args: args{rangeStr: "[01-04/2]"},
			want: []string{"01", "03"},
		},
		{
			name: "Step, range value with zero padding of two digits, but with 3 digit end value",
			args: args{rangeStr: "[01-150/50]"},
			want: []string{"01", "51", "101"},
		},
		{
			name: "Overlapping union ranges with padding",
			args: args{rangeStr: "[01-02,01-04]"},
			want: []string{"01", "02", "03", "04"},
		},
		{
			name: "Overlapping union ranges with inconsistent padding",
			args: args{rangeStr: "[01-02,001-004]"},
			want: []string{"01", "001", "02", "002", "003", "004"},
		},
		{
			name:    "Missing brackets",
			args:    args{rangeStr: "1"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Missing left bracket",
			args:    args{rangeStr: "1]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Missing right bracket",
			args:    args{rangeStr: "[1"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Step without range",
			args:    args{rangeStr: "[1/2]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Single value range non-integer",
			args:    args{rangeStr: "[a]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Range starts with non-integer",
			args:    args{rangeStr: "[a-2]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Range ends with non-integer",
			args:    args{rangeStr: "[1-b]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Range start value is greater than the end value",
			args:    args{rangeStr: "[2-1]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Range start value has zero padding greater than the end value",
			args:    args{rangeStr: "[001-2]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Range value with zero padding on end value of great length than start value",
			args:    args{rangeStr: "[01-004]"},
			want:    []string{},
			wantErr: true,
		},
		{
			name:    "Two step delineator error, passing error up from parseStep",
			args:    args{rangeStr: "[1-4//2]"},
			want:    []string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRange(tt.args.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseStep(t *testing.T) {
	type args struct {
		rangeStr string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   uint64
		wantErr bool
	}{
		{
			name: "Empty",
			want: "",
		},
		{
			name: "No step, single value",
			args: args{rangeStr: "1"},
			want: "1",
		},
		{
			name:  "No step, range value",
			args:  args{rangeStr: "1-2"},
			want:  "1-2",
			want1: 0,
		},
		{
			name:  "Step, range value",
			args:  args{rangeStr: "1-4/2"},
			want:  "1-4",
			want1: 2,
		},
		{
			name:    "Two step delineator error",
			args:    args{rangeStr: "1-2//2"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Non-integer step error",
			args:    args{rangeStr: "1-2/a"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseStep(tt.args.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseStep() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseStep() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
