package db

import "testing"

func TestNormalizeError(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"TypeError: Cannot read at line:42", "typeerror: cannot read at line"},
		{"panic at 0x7fff1234 in main", "panic at 0xaddr in main"},
		{"  multiple   spaces  ", "multiple spaces"},
	}
	for _, c := range cases {
		got := NormalizeError(c.in)
		if got != c.want {
			t.Errorf("NormalizeError(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHashError(t *testing.T) {
	// same logical error despite line number difference → same hash
	h1 := HashError("TypeError: Cannot read at line:42")
	h2 := HashError("TypeError: Cannot read at line:99")
	if h1 != h2 {
		t.Errorf("hashes differ for same normalized error: %s vs %s", h1, h2)
	}
}
