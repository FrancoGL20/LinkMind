package service

import "testing"

// TestEncode verifies that Encode produces the expected Base62 strings.
func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{name: "zero", input: 0, expected: "0"},
		{name: "one", input: 1, expected: "1"},
		{name: "sixty-one (max single digit)", input: 61, expected: "z"},
		{name: "sixty-two (first two-digit)", input: 62, expected: "10"},
		{name: "known value 1024", input: 1024, expected: "GW"},
		{name: "large number", input: 56_800_235_583, expected: "zzzzzz"}, // 62^6 - 1
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Encode(tc.input)
			if got != tc.expected {
				t.Errorf("Encode(%d) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestDecode verifies that Decode correctly converts Base62 strings back to int64.
func TestDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{name: "zero string", input: "0", expected: 0},
		{name: "one string", input: "1", expected: 1},
		{name: "max single digit", input: "z", expected: 61},
		{name: "first two-digit", input: "10", expected: 62},
		{name: "known value GW", input: "GW", expected: 1024},
		{name: "six-digit max", input: "zzzzzz", expected: 56_800_235_583},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Decode(tc.input)
			if got != tc.expected {
				t.Errorf("Decode(%q) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

// TestDecodeInvalidInput verifies that Decode returns -1 for invalid inputs.
func TestDecodeInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "contains space", input: "g W"},
		{name: "contains special char", input: "g@W"},
		{name: "contains dash", input: "g-W"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Decode(tc.input)
			if got != -1 {
				t.Errorf("Decode(%q) = %d, want -1 for invalid input", tc.input, got)
			}
		})
	}
}

// TestEncodeDecodeRoundTrip verifies that Decode(Encode(n)) == n for a set of values.
func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []int64{0, 1, 62, 1024, 999_999, 1_000_000_000, 56_800_235_583}

	for _, n := range cases {
		encoded := Encode(n)
		decoded := Decode(encoded)
		if decoded != n {
			t.Errorf("round-trip failed: Decode(Encode(%d)) = %d (via %q)", n, decoded, encoded)
		}
	}
}
