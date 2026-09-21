package pyin

import (
	"slices"
	"testing"
)

func TestKeys(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"杭州备份机", []string{"hangzhoubeifenji", "hzbfj"}},
		{"广州生产1", []string{"guangzhoushengchan1", "gzsc1"}},
		{"北京GPU", []string{"beijinggpu", "bjgpu"}}, // mixed Han + ASCII
		{"zgocloud", nil}, // pure ASCII costs nothing
		{"", nil},
	}
	for _, tt := range tests {
		got := Keys(tt.in)
		if !slices.Equal(got, tt.want) {
			t.Errorf("Keys(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
