package domain

import "testing"

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "Alice", want: "alice"},
		{in: "  Bob  ", want: "bob"},
		{in: "CHARLIE   brown", want: "charlie brown"},
		{in: "Ärger Öde Übel", want: "ärger öde übel"},
		{in: "a\tb\nc", want: "a b c"},
		{in: "", want: ""},
		{in: "   ", want: ""},
		{in: "O'Brien", want: "o'brien"},
	}
	for _, tt := range tests {
		if got := NormalizeName(tt.in); got != tt.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "plain", in: "Alice", want: "Alice"},
		{name: "trimmed", in: "  Alice ", want: "Alice"},
		{name: "empty", in: "", wantErr: true},
		{name: "whitespace only", in: "   ", wantErr: true},
		{name: "max length ok", in: string(makeRunes(80)), want: string(makeRunes(80))},
		{name: "too long", in: string(makeRunes(81)), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateName(tt.in)
			if tt.wantErr != (err != nil) {
				t.Fatalf("ValidateName(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ValidateName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func makeRunes(n int) []rune {
	out := make([]rune, n)
	for i := range out {
		out[i] = 'a'
	}
	return out
}
