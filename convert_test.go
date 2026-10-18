package loopia

import "testing"

func TestToBool(t *testing.T) {
	tests := []struct {
		in   any
		want bool
	}{
		{true, true}, {false, false},
		{int64(1), true}, {int64(0), false},
		{1.0, true}, {0.0, false},
		{"1", true}, {"true", true}, {"0", false}, {"nope", false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := toBool(tt.in); got != tt.want {
			t.Errorf("toBool(%#v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		in   any
		want float64
	}{
		{1.5, 1.5}, {int64(3), 3}, {"2.25", 2.25}, {"x", 0}, {nil, 0},
	}
	for _, tt := range tests {
		if got := toFloat(tt.in); got != tt.want {
			t.Errorf("toFloat(%#v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{nil, ""}, {"abc", "abc"}, {int64(42), "42"}, {12.5, "12.5"}, {true, "true"},
	}
	for _, tt := range tests {
		if got := toString(tt.in); got != tt.want {
			t.Errorf("toString(%#v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStatusErrorMessage(t *testing.T) {
	if got := ErrAuth.Error(); got != "loopia: AUTH_ERROR" {
		t.Errorf("Error() = %q", got)
	}
}
