package tabledriven

import "testing"

func TestGreet(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "Alice", want: "Hello, Alice!"},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Greet(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Greet(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{input: 5, want: 5},
		{input: -3, want: 3},
		{input: 0, want: 0},
	}

	for _, tt := range tests {
		got := Abs(tt.input)
		if got != tt.want {
			t.Errorf("Abs(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
