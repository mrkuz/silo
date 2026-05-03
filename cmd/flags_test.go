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
		wantSV  *bool
		wantErr bool
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantP:   nil,
			wantSV:  nil,
			wantErr: false,
		},
		{
			name:    "--podman",
			args:    []string{"--podman"},
			wantP:   ptr(true),
			wantSV:  nil,
			wantErr: false,
		},
		{
			name:    "--no-podman",
			args:    []string{"--no-podman"},
			wantP:   ptr(false),
			wantSV:  nil,
			wantErr: false,
		},
		{
			name:    "--shared-volume",
			args:    []string{"--shared-volume"},
			wantP:   nil,
			wantSV:  ptr(true),
			wantErr: false,
		},
		{
			name:    "--no-shared-volume",
			args:    []string{"--no-shared-volume"},
			wantP:   nil,
			wantSV:  ptr(false),
			wantErr: false,
		},
		{
			name:    "--podman --shared-volume",
			args:    []string{"--podman", "--shared-volume"},
			wantP:   ptr(true),
			wantSV:  ptr(true),
			wantErr: false,
		},
		{
			name:    "--podman --no-shared-volume",
			args:    []string{"--podman", "--no-shared-volume"},
			wantP:   ptr(true),
			wantSV:  ptr(false),
			wantErr: false,
		},
		{
			name:    "last flag wins: --podman --no-podman",
			args:    []string{"--podman", "--no-podman"},
			wantP:   ptr(false),
			wantSV:  nil,
			wantErr: false,
		},
		{
			name:    "last flag wins: --shared-volume --no-shared-volume",
			args:    []string{"--shared-volume", "--no-shared-volume"},
			wantP:   nil,
			wantSV:  ptr(false),
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
			if tt.wantSV == nil && f.SharedVolume != nil {
				t.Errorf("expected SharedVolume=nil, got %v", *f.SharedVolume)
			}
			if tt.wantSV != nil && f.SharedVolume == nil {
				t.Errorf("expected SharedVolume=%v, got nil", *tt.wantSV)
			}
			if tt.wantSV != nil && f.SharedVolume != nil && *f.SharedVolume != *tt.wantSV {
				t.Errorf("expected SharedVolume=%v, got %v", *tt.wantSV, *f.SharedVolume)
			}
		})
	}
}

func TestParseRemoveFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantF   bool
		wantErr bool
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantF:   false,
			wantErr: false,
		},
		{
			name:    "-f",
			args:    []string{"-f"},
			wantF:   true,
			wantErr: false,
		},
		{
			name:    "--force",
			args:    []string{"--force"},
			wantF:   true,
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
			f, err := cmd.ParseRemoveFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f != tt.wantF {
				t.Errorf("expected force=%v, got %v", tt.wantF, f)
			}
		})
	}
}

func TestParseForceFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantForce bool
		wantRest  []string
	}{
		{
			name:      "no flags",
			args:      []string{},
			wantForce: false,
			wantRest:  nil,
		},
		{
			name:      "-f",
			args:      []string{"-f"},
			wantForce: true,
			wantRest:  nil,
		},
		{
			name:      "--force",
			args:      []string{"--force"},
			wantForce: true,
			wantRest:  nil,
		},
		{
			name:      "-f with remaining args",
			args:      []string{"-f", "--podman"},
			wantForce: true,
			wantRest:  []string{"--podman"},
		},
		{
			name:      "unknown flag preserved",
			args:      []string{"--podman", "--unknown"},
			wantForce: false,
			wantRest:  []string{"--podman", "--unknown"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			force, rest := cmd.ParseForceFlag(tt.args)
			if force != tt.wantForce {
				t.Errorf("ParseForceFlag(%v): force=%v, want %v", tt.args, force, tt.wantForce)
			}
			if len(rest) != len(tt.wantRest) {
				t.Errorf("ParseForceFlag(%v): rest=%v, want %v", tt.args, rest, tt.wantRest)
				return
			}
			for i := range rest {
				if rest[i] != tt.wantRest[i] {
					t.Errorf("ParseForceFlag(%v): rest[%d]=%v, want %v", tt.args, i, rest[i], tt.wantRest[i])
				}
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
