package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Pattern matching tests

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		key     string
		pattern string
		want    bool
	}{
		// Exact matches
		{"podman build foo", "podman build foo", true},
		{"podman build foo", "podman build bar", false},
		// <any> - exactly one token
		{"podman image exists foo", "podman image exists <any>", true},
		{"podman build", "podman <any>", true},
		// <any?> - zero or one token
		{"podman build", "podman build <any?>", true},
		{"podman build foo", "podman build <any?>", true},
		{"podman", "podman <any?>", true},
		// <...> - one or more tokens
		{"podman build foo", "podman <...>", true},
		{"podman build foo bar", "podman <...>", true},
		{"podman", "podman <...>", false},
		// <...? > - zero or more tokens
		{"podman", "podman <...?>", true},
		{"podman build foo", "podman <...?>", true},
		// Additional cases
		{"podman build silo-alice", "podman build <any>", true},
		{"podman build silo-alice", "podman <...>", true},
	}

	for _, tt := range tests {
		if got := matchPattern(tt.key, tt.pattern); got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.key, tt.pattern, got, tt.want)
		}
	}
}

func TestMatchPatternEdgeCases(t *testing.T) {
	tests := []struct {
		key     string
		pattern string
		want    bool
	}{
		{"", "", true},
		{"", "<...?>", true},
		{"", "<any?>", true},
		{"", "<any>", false},
		{"", "<...>", false},
		{"a", "<any>", true},
		{"a", "<any?>", true},
		{"a b", "<any>", false},
		{"a b", "<any?> <any?>", true},
		{"a", "<any> <any?>", true},
		{"a b", "<any> <any?>", true},
	}
	for _, tt := range tests {
		if got := matchPattern(tt.key, tt.pattern); got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.key, tt.pattern, got, tt.want)
		}
	}
}

func TestPatternVariants(t *testing.T) {
	tests := []struct {
		key     string
		pattern string
		want    bool
	}{
		{"podman build", "podman <any>", true},
		{"podman build foo", "podman <any>", false},
		{"podman build foo bar", "podman <any>", false},
		{"podman", "podman <any?>", true},
		{"podman build", "podman <any?>", true},
		{"podman build foo", "podman <any?>", false},
		{"podman build", "podman <...>", true},
		{"podman build foo", "podman <...>", true},
		{"podman", "podman <...>", false},
		{"podman", "podman <...?>", true},
		{"podman build", "podman <...?>", true},
		{"podman build foo", "podman <...?>", true},
	}

	for _, tt := range tests {
		r := ExecRecord{Name: "podman", Args: strings.Fields(tt.key)[1:]}
		if got := r.Match(tt.pattern); got != tt.want {
			t.Errorf("ExecRecord{Args:%v}.Match(%q) = %v, want %v", r.Args, tt.pattern, got, tt.want)
		}
	}
}

// Record String() tests

func TestExecKey(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"podman", []string{"build", "foo"}, "podman build foo"},
		{"ls", []string{"-la"}, "ls -la"},
		{"echo", []string{}, "echo"},
	}

	for _, tt := range tests {
		if got := execKey(tt.name, tt.args); got != tt.want {
			t.Errorf("execKey(%q, %v) = %q, want %q", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestRecordString(t *testing.T) {
	t.Run("ExecRecord", func(t *testing.T) {
		r := ExecRecord{Name: "podman", Args: []string{"build", "-t", "silo-alice"}}
		want := "podman build -t silo-alice"
		if got := r.String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("ReadRecord", func(t *testing.T) {
		r := ReadRecord{Path: "/etc/silo/home.nix"}
		want := "read(/etc/silo/home.nix)"
		if got := r.String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("WriteRecord", func(t *testing.T) {
		r := WriteRecord{Path: "/tmp/file", Content: []byte("hello\nworld\n")}
		want := "write(/tmp/file, hello\nworld\n)"
		if got := r.String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})
}

// Record Match() tests

func TestExecRecordMatch(t *testing.T) {
	r := ExecRecord{Name: "podman", Args: []string{"build", "silo-alice"}}

	if !r.Match("podman", "build", "silo-alice") {
		t.Error("expected exact match podman build silo-alice")
	}

	if r.Match("podman", "build") {
		t.Error("should not match - extra tokens leftover")
	}

	if r.Match("podman", "rm") {
		t.Error("did not expect match for podman rm")
	}
	if r.Match("nonexistent") {
		t.Error("did not expect match for nonexistent")
	}
	if r.Match("podman", "build", "silo-bob") {
		t.Error("did not expect match for podman build silo-bob")
	}
}

func TestReadRecordMatch(t *testing.T) {
	r := ReadRecord{Path: "/etc/silo/home.nix"}

	if !r.Match("/etc/silo/home.nix") {
		t.Error("expected exact match")
	}

	if r.Match("/etc/silo") {
		t.Error("should not match - different paths")
	}

	if r.Match("/nonexistent") {
		t.Error("did not expect match for /nonexistent")
	}
}

func TestWriteRecordMatch(t *testing.T) {
	r := WriteRecord{Path: "/tmp/output.txt"}

	if !r.Match("/tmp/output.txt") {
		t.Error("expected exact match")
	}

	if r.Match("/tmp") {
		t.Error("should not match - different paths")
	}

	if r.Match("/nonexistent") {
		t.Error("did not expect match for /nonexistent")
	}
}

// Mock tests

func TestMockExec(t *testing.T) {
	mock := NewMock(t)

	responses := map[string]*exec.Cmd{
		"podman build silo-alice":   exec.Command("true"),
		"podman image exists <any>": exec.Command("false"),
	}
	mock.MockExec(responses)

	cmd := ExecCommand("podman", "build", "silo-alice")
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}

	cmd = ExecCommand("podman", "image", "exists", "silo-anything")
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}

	if len(mock.execCalls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(mock.execCalls))
	}

	if mock.execCalls[0].Seq <= 0 {
		t.Error("expected seq > 0")
	}
}

func TestMockRead(t *testing.T) {
	mock := NewMock(t)

	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("test content"), 0644)

	responses := map[string][]byte{
		path: []byte("mocked content"),
	}
	mock.MockRead(responses)

	content, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "mocked content" {
		t.Errorf("expected 'mocked content', got %q", string(content))
	}

	if len(mock.readCalls) != 1 {
		t.Errorf("expected 1 call, got %d", len(mock.readCalls))
	}

	if mock.readCalls[0].Path != path {
		t.Errorf("expected path %q, got %q", path, mock.readCalls[0].Path)
	}

	if string(mock.readCalls[0].Content) != "mocked content" {
		t.Errorf("expected content 'mocked content', got %q", string(mock.readCalls[0].Content))
	}
}

func TestMockWrite(t *testing.T) {
	mock := NewMock(t)
	mock.MockWrite()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.txt")

	err := WriteFile(path, []byte("hello world"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if len(mock.writeCalls) != 1 {
		t.Errorf("expected 1 call, got %d", len(mock.writeCalls))
	}

	if mock.writeCalls[0].Path != path {
		t.Errorf("expected path %q, got %q", path, mock.writeCalls[0].Path)
	}

	if string(mock.writeCalls[0].Content) != "hello world" {
		t.Errorf("expected content 'hello world', got %q", string(mock.writeCalls[0].Content))
	}
}

func TestMockReadContent(t *testing.T) {
	mock := NewMock(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("original"), 0644)

	responses := map[string][]byte{path: []byte("mocked")}
	mock.MockRead(responses)

	ReadFile(path)

	rec := mock.AssertRead(path)
	if rec == nil {
		t.Fatal("expected read record")
	}
	if string(rec.Content) != "mocked" {
		t.Errorf("expected content 'mocked', got %q", string(rec.Content))
	}
}

func TestMockWriteReadContent(t *testing.T) {
	mock := NewMock(t)
	mock.MockRead(map[string][]byte{"/path": []byte("mocked")})
	mock.MockWrite()

	WriteFile("/path", []byte("output"))

	rec := mock.AssertWrite("/path")
	if rec == nil {
		t.Fatal("expected write record")
	}
	if string(rec.Content) != "output" {
		t.Errorf("expected content 'output', got %q", string(rec.Content))
	}
}

// Assert tests

func TestAssertExec(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman build <any>": exec.Command("true"),
		})

		ExecCommand("podman", "build", "silo-alice")

		rec := mock.AssertExec("podman", "build", "silo-alice")
		if rec == nil {
			t.Fatal("expected record, got nil")
		}
		if rec.Name != "podman" {
			t.Errorf("expected name 'podman', got %q", rec.Name)
		}
		if rec.Args[1] != "silo-alice" {
			t.Errorf("expected args[1] 'silo-alice', got %q", rec.Args[1])
		}
	})

	t.Run("notFound", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{})
		if len(mock.execCalls) != 0 {
			t.Errorf("expected no calls, got %d", len(mock.execCalls))
		}
	})
}

func TestAssertNoExec(t *testing.T) {
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{
		"podman build <any>": exec.Command("true"),
	})

	ExecCommand("podman", "build", "silo-alice")

	mock.AssertNoExec("podman", "rm", "<...>")
}

func TestAssertRead(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		mock := NewMock(t)
		tmp := t.TempDir()
		path := filepath.Join(tmp, "test.txt")
		os.WriteFile(path, []byte("original"), 0644)

		mock.MockRead(map[string][]byte{path: []byte("mocked")})

		ReadFile(path)

		rec := mock.AssertRead(path)
		if rec == nil {
			t.Fatal("expected record, got nil")
		}
		if rec.Path != path {
			t.Errorf("expected path %q, got %q", path, rec.Path)
		}
	})

	t.Run("notFound", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockRead(map[string][]byte{})
		if len(mock.readCalls) != 0 {
			t.Errorf("expected no calls, got %d", len(mock.readCalls))
		}
	})
}

func TestAssertWrite(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockWrite()

		tmp := t.TempDir()
		path := filepath.Join(tmp, "out.txt")

		WriteFile(path, []byte("hello"))

		rec := mock.AssertWrite(path)
		if rec == nil {
			t.Fatal("expected record, got nil")
		}
		if string(rec.Content) != "hello" {
			t.Errorf("expected content 'hello', got %q", string(rec.Content))
		}
	})

	t.Run("notFound", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockWrite()
		if len(mock.writeCalls) != 0 {
			t.Errorf("expected no calls, got %d", len(mock.writeCalls))
		}
	})
}

func TestAssertNoRead(t *testing.T) {
	mock := NewMock(t)
	mock.MockRead(map[string][]byte{"/path": []byte("mocked")})
	ReadFile("/path")
	mock.AssertNoRead("/other")
}

func TestAssertNoWrite(t *testing.T) {
	mock := NewMock(t)
	mock.MockWrite()
	WriteFile("/path", []byte("data"))
	mock.AssertNoWrite("/other")
}

// Sequence and ordering tests

func TestSeqOrdering(t *testing.T) {
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{"<...? >": exec.Command("true")})
	mock.MockRead(map[string][]byte{})
	mock.MockWrite()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("data"), 0644)

	ExecCommand("podman", "build", "user")
	firstSeq := mock.execCalls[0].Seq

	ReadFile(path)
	readSeq := mock.readCalls[0].Seq

	WriteFile(path, []byte("output"))
	writeSeq := mock.writeCalls[0].Seq

	if readSeq <= firstSeq {
		t.Error("read seq should be after exec seq")
	}

	if writeSeq <= readSeq {
		t.Error("write seq should be after read seq")
	}
}

func TestAssertExecMultiple(t *testing.T) {
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{"<...? >": exec.Command("true")})

	ExecCommand("podman", "build", "first")
	ExecCommand("podman", "build", "second")

	rec := mock.AssertExec("podman", "build", "second")
	if rec == nil {
		t.Fatal("expected record")
	}
	if rec.Args[1] != "second" {
		t.Errorf("expected 'second', got %q", rec.Args[1])
	}
	if rec.Seq != mock.execCalls[1].Seq {
		t.Error("should return second record")
	}

	recFirst := mock.AssertExec("podman", "build", "first")
	if recFirst == nil {
		t.Fatal("expected record")
	}
	if recFirst.Args[1] != "first" {
		t.Errorf("expected 'first', got %q", recFirst.Args[1])
	}
}

// Pattern matching with mocks

func TestMultiArgPattern(t *testing.T) {
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{
		"podman <any> <any> <any>": exec.Command("true"),
	})

	ExecCommand("podman", "image", "exists", "silo-test")

	if len(mock.execCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.execCalls))
	}

	rec := mock.AssertExec("podman", "image", "exists", "silo-test")
	if rec == nil {
		t.Error("expected record")
	}
}

func TestEllipsisOneOrMore(t *testing.T) {
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{
		"podman build <...>": exec.Command("true"),
	})

	ExecCommand("podman", "build", "silo-alice")
	ExecCommand("podman", "build", "silo-bob")

	if len(mock.execCalls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mock.execCalls))
	}

	rec := mock.AssertExec("podman", "build", "silo-alice")
	if rec == nil {
		t.Fatal("expected record for silo-alice")
	}

	rec2 := mock.AssertExec("podman", "build", "silo-bob")
	if rec2 == nil {
		t.Fatal("expected record for silo-bob")
	}
}

// Seams override test

func TestSeamsOverrideable(t *testing.T) {
	orig := ExecCommand
	defer func() { ExecCommand = orig }()

	customCalled := false
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		customCalled = true
		return exec.Command("true")
	}

	ExecCommand("test", "arg")
	if !customCalled {
		t.Error("expected custom ExecCommand to be called")
	}
}
