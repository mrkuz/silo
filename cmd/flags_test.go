package cmd_test

import (
	"testing"

	"github.com/mrkuz/silo/cmd"
)

func TestParseRunFlags(t *testing.T) {
	tests := []struct {
		args     []string
		wantStop bool
		wantRemove  bool
		wantErr  bool
	}{
		{[]string{}, false, false, false},
		{[]string{"--stop"}, true, false, false},
		{[]string{"--rm"}, true, true, false},
		{[]string{"--unknown"}, false, false, true},
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
		if f.Stop != tt.wantStop || f.Remove != tt.wantRemove {
			t.Errorf("ParseRunFlags(%v) = {Stop:%v Rmi:%v}, want {Stop:%v Rmi:%v}",
				tt.args, f.Stop, f.Remove, tt.wantStop, tt.wantRemove)
		}
	}
}

func TestParseRunFlagsExtra(t *testing.T) {
	f, err := cmd.ParseRunFlags([]string{"--", "arg1", "arg2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Extra) != 2 || f.Extra[0] != "arg1" || f.Extra[1] != "arg2" {
		t.Errorf("expected Extra=[arg1 arg2], got %v", f.Extra)
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

func TestParseCreateFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantD   bool
		wantErr bool
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantD:   false,
			wantErr: false,
		},
		{
			name:    "--dry-run",
			args:    []string{"--dry-run"},
			wantD:   true,
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
			f, err := cmd.ParseCreateFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f.DryRun != tt.wantD {
				t.Errorf("expected DryRun=%v, got %v", tt.wantD, f.DryRun)
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

func ptr[T any](v T) *T {
	return &v
}
