package cmd_test

import (
	"testing"

	"github.com/mrkuz/silo/cmd"
)

func TestParseRunFlags(t *testing.T) {
	tests := []struct {
		args     []string
		wantStop bool
		wantErr  bool
	}{
		{[]string{}, false, false},
		{[]string{"--stop"}, true, false},
		{[]string{"--unknown"}, false, true},
	}
	for _, tt := range tests {
		f, err := cmd.ParseRunFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseRunFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRunFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if f.Stop != tt.wantStop {
			t.Errorf("ParseRunFlags(%v).Stop = %v, want %v", tt.args, f.Stop, tt.wantStop)
		}
	}
}

func TestParseRunFlagsExtra(t *testing.T) {
	_, err := cmd.ParseRunFlags([]string{"arg1", "arg2"})
	if err == nil {
		t.Error("expected error for extra arguments")
	}
}
