package reverse

import (
	"fmt"
	"testing"
)

func TestReverseRunes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ascii", "Go", "oG"},
		{"zh", "小册", "册小"},
		{"empty", "", ""},
		{"mixed", "Go小册", "册小oG"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReverseRunes(tt.in); got != tt.want {
				t.Fatalf("ReverseRunes(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestReverseTwice(t *testing.T) {
	s := "Go小册👋"
	if got := ReverseRunes(ReverseRunes(s)); got != s {
		t.Fatalf("got %q", got)
	}
}

func FuzzReverseRunes(f *testing.F) {
	f.Add("Go小册")
	f.Fuzz(func(t *testing.T, s string) {
		if got := ReverseRunes(ReverseRunes(s)); got != s {
			t.Fatalf("%q -> %q", s, got)
		}
	})
}

func ExampleReverseRunes() {
	fmt.Println(ReverseRunes("册小"))
	// Output:
	// 小册
}
