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

func TestParseInitFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantP   *bool
		wantErr bool
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantP:   nil,
			wantErr: false,
		},
		{
			name:    "--podman",
			args:    []string{"--podman"},
			wantP:   ptr(true),
			wantErr: false,
		},
		{
			name:    "--no-podman",
			args:    []string{"--no-podman"},
			wantP:   ptr(false),
			wantErr: false,
		},
		{
			name:    "last flag wins: --podman --no-podman",
			args:    []string{"--podman", "--no-podman"},
			wantP:   ptr(false),
			wantErr: false,
		},
		{
			name:    "unknown flag",
			args:    []string{"--unknown"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := cmd.ParseInitFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantP == nil && f.Podman != nil {
				t.Errorf("expected Podman=nil, got %v", *f.Podman)
			}
			if tt.wantP != nil && f.Podman == nil {
				t.Errorf("expected Podman=%v, got nil", *tt.wantP)
			}
			if tt.wantP != nil && f.Podman != nil && *f.Podman != *tt.wantP {
				t.Errorf("expected Podman=%v, got %v", *tt.wantP, *f.Podman)
			}
		})
	}
}

func TestParseRebuildAndNoCache(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantRebuild bool
		wantNoCache bool
		wantRest  []string
		wantErr   bool
	}{
		{
			name:      "no flags",
			args:      []string{},
			wantRebuild: false,
			wantNoCache: false,
			wantRest:  nil,
			wantErr:   false,
		},
		{
			name:      "--rebuild",
			args:      []string{"--rebuild"},
			wantRebuild: true,
			wantNoCache: false,
			wantRest:  nil,
			wantErr:   false,
		},
		{
			name:      "--no-cache",
			args:      []string{"--no-cache"},
			wantRebuild: false,
			wantNoCache: true,
			wantRest:  nil,
			wantErr:   false,
		},
		{
			name:      "--rebuild --no-cache",
			args:      []string{"--rebuild", "--no-cache"},
			wantRebuild: true,
			wantNoCache: true,
			wantRest:  nil,
			wantErr:   false,
		},
		{
			name:      "unknown flag returns error",
			args:      []string{"--unknown"},
			wantRebuild: false,
			wantNoCache: false,
			wantRest:  nil,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rebuild, noCache, rest, err := cmd.ParseRebuildAndNoCache("test", tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRebuildAndNoCache(%v): expected error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRebuildAndNoCache(%v): unexpected error: %v", tt.args, err)
			}
			if rebuild != tt.wantRebuild {
				t.Errorf("ParseRebuildAndNoCache(%v): rebuild=%v, want %v", tt.args, rebuild, tt.wantRebuild)
			}
			if noCache != tt.wantNoCache {
				t.Errorf("ParseRebuildAndNoCache(%v): noCache=%v, want %v", tt.args, noCache, tt.wantNoCache)
			}
			if len(rest) != len(tt.wantRest) {
				t.Errorf("ParseRebuildAndNoCache(%v): rest=%v, want %v", tt.args, rest, tt.wantRest)
				return
			}
			for i := range rest {
				if rest[i] != tt.wantRest[i] {
					t.Errorf("ParseRebuildAndNoCache(%v): rest[%d]=%v, want %v", tt.args, i, rest[i], tt.wantRest[i])
				}
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
