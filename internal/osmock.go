package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Seams for OS operations - override in tests.
var (
	ExecCommand = exec.Command
	ReadFile    = os.ReadFile
	WriteFile   = func(path string, content []byte) error { return os.WriteFile(path, content, 0644) }
)

// Pattern matching syntax for exec:
//   <any>   - matches exactly one token
//   <any?>  - matches zero or one token
//   <...>   - matches one or more tokens
//   <...? >  - matches zero or more tokens
// Tokens are separated by whitespace.

// matchPattern matches key against pattern using glob-style placeholders.
// Returns true if the entire key matches the pattern.
func matchPattern(key, pattern string) bool {
	if pattern == "" {
		return key == ""
	}

	keyTokens := strings.Fields(key)
	patternTokens := strings.Fields(pattern)

	return matchTokens(keyTokens, patternTokens)
}

// matchTokens recursively matches key tokens against pattern tokens.
// Returns true only if all pattern tokens are matched AND all key tokens are consumed.
func matchTokens(key, pattern []string) bool {
	if len(pattern) == 0 {
		return len(key) == 0
	}

	p := pattern[0]

	if p == "<...>" {
		if len(key) == 0 {
			return false
		}
		for i := 1; i <= len(key); i++ {
			if matchTokens(key[i:], pattern[1:]) {
				return true
			}
		}
		return false
	}

	if p == "<...?>" {
		for i := 0; i <= len(key); i++ {
			if matchTokens(key[i:], pattern[1:]) {
				return true
			}
		}
		return false
	}

	if p == "<any>" {
		if len(key) == 0 {
			return false
		}
		return matchTokens(key[1:], pattern[1:])
	}

	if p == "<any?>" {
		if matchTokens(key, pattern[1:]) {
			return true
		}
		if len(key) > 0 {
			return matchTokens(key[1:], pattern[1:])
		}
		return false
	}

	if len(key) == 0 {
		return false
	}
	if key[0] == p {
		return matchTokens(key[1:], pattern[1:])
	}
	return false
}

// ExecRecord records a single exec operation.
type ExecRecord struct {
	Seq  int
	Name string
	Args []string
}

// Match returns true if the exec command matches the given pattern tokens.
func (r *ExecRecord) Match(pattern ...string) bool {
	target := r.Name + " " + strings.Join(r.Args, " ")
	return matchPattern(target, strings.Join(pattern, " "))
}

func (r *ExecRecord) String() string {
	return r.Name + " " + strings.Join(r.Args, " ")
}

// ReadRecord records a single read operation.
type ReadRecord struct {
	Seq     int
	Path    string
	Content []byte
}

// Match returns true if the path exactly matches the given path.
func (r *ReadRecord) Match(path string) bool {
	return r.Path == path
}

func (r *ReadRecord) String() string {
	return "read(" + r.Path + ")"
}

// WriteRecord records a single write operation.
type WriteRecord struct {
	Seq     int
	Path    string
	Content []byte
}

// Match returns true if the path exactly matches the given path.
func (r *WriteRecord) Match(path string) bool {
	return r.Path == path
}

func (r *WriteRecord) String() string {
	return "write(" + r.Path + ", " + string(r.Content) + ")"
}

// execKey builds the command key from name and args.
func execKey(name string, args []string) string {
	all := append([]string{name}, args...)
	return strings.Join(all, " ")
}

// Mock provides mocking for exec, read, and write operations.
type Mock struct {
	t *testing.T

	seq        int
	execCalls  []ExecRecord
	readCalls  []ReadRecord
	writeCalls []WriteRecord
	execReturn map[string]*exec.Cmd
	readReturn map[string][]byte
}

// NewMock creates a Mock bound to the given test.
func NewMock(t *testing.T) *Mock {
	return &Mock{
		t:          t,
		execReturn: make(map[string]*exec.Cmd),
		readReturn: make(map[string][]byte),
	}
}

func (m *Mock) nextSeq() int {
	m.seq++
	return m.seq
}

// Reset clears all recorded calls but preserves mock responses.
func (m *Mock) Reset() {
	m.execCalls = nil
	m.readCalls = nil
	m.writeCalls = nil
}

// MockExec installs mock execCommand.responses maps full command string to *exec.Cmd.
// Patterns use glob-style matching: <any>, <any?>, <...>, <...? >
func (m *Mock) MockExec(responses map[string]*exec.Cmd) {
	m.execReturn = responses
	orig := ExecCommand
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		record := ExecRecord{
			Seq:  m.nextSeq(),
			Name: name,
			Args: args,
		}
		m.execCalls = append(m.execCalls, record)

		key := execKey(name, args)
		if cmd, ok := m.execReturn[key]; ok {
			return cmd
		}
		for pattern, cmd := range m.execReturn {
			if matchPattern(key, pattern) {
				return cmd
			}
		}
		return exec.Command("true")
	}
	m.t.Cleanup(func() { ExecCommand = orig })
}

// MockRead installs mock readFile.responses maps path to content.
func (m *Mock) MockRead(responses map[string][]byte) {
	m.readReturn = responses
	orig := ReadFile
	ReadFile = func(path string) ([]byte, error) {
		var content []byte
		if c, ok := m.readReturn[path]; ok {
			content = c
		} else {
			var err error
			content, err = orig(path)
			if err != nil {
				return nil, err
			}
		}
		record := ReadRecord{
			Seq:     m.nextSeq(),
			Path:    path,
			Content: content,
		}
		m.readCalls = append(m.readCalls, record)
		return content, nil
	}
	m.t.Cleanup(func() { ReadFile = orig })
}

// MockWrite installs mock writeFile that records all writes.
func (m *Mock) MockWrite() {
	orig := WriteFile
	WriteFile = func(path string, content []byte) error {
		record := WriteRecord{
			Seq:     m.nextSeq(),
			Path:    path,
			Content: content,
		}
		m.writeCalls = append(m.writeCalls, record)
		return orig(path, content)
	}
	m.t.Cleanup(func() { WriteFile = orig })
}

// AssertExec finds the first exec call matching the pattern.
// Returns the record if found, or nil and fails the test if not found.
func (m *Mock) AssertExec(pattern ...string) *ExecRecord {
	for i := range m.execCalls {
		if m.execCalls[i].Match(pattern...) {
			return &m.execCalls[i]
		}
	}
	m.t.Errorf("expected exec call matching %v", pattern)
	return nil
}

// AssertNoExec fails the test if any exec call matches the pattern.
func (m *Mock) AssertNoExec(pattern ...string) {
	for i := range m.execCalls {
		if m.execCalls[i].Match(pattern...) {
			m.t.Errorf("expected no exec call matching %v", pattern)
			return
		}
	}
}

// AssertRead finds the first read call matching the exact path.
// Returns the record if found, or nil and fails the test if not found.
func (m *Mock) AssertRead(path string) *ReadRecord {
	for i := range m.readCalls {
		if m.readCalls[i].Match(path) {
			return &m.readCalls[i]
		}
	}
	m.t.Errorf("expected read call matching %s", path)
	return nil
}

// AssertNoRead fails the test if any read call matches the path.
func (m *Mock) AssertNoRead(path string) {
	for i := range m.readCalls {
		if m.readCalls[i].Match(path) {
			m.t.Errorf("expected no read call matching %s", path)
			return
		}
	}
}

// AssertWrite finds the first write call matching the exact path.
// Returns the record if found, or nil and fails the test if not found.
func (m *Mock) AssertWrite(path string) *WriteRecord {
	for i := range m.writeCalls {
		if m.writeCalls[i].Match(path) {
			return &m.writeCalls[i]
		}
	}
	m.t.Errorf("expected write call matching %s", path)
	return nil
}

// AssertNoWrite fails the test if any write call matches the path.
func (m *Mock) AssertNoWrite(path string) {
	for i := range m.writeCalls {
		if m.writeCalls[i].Match(path) {
			m.t.Errorf("expected no write call matching %s", path)
			return
		}
	}
}

// Dump prints all recorded calls for debugging.
func (m *Mock) Dump() {
	fmt.Println("=== Exec Calls ===")
	for i, c := range m.execCalls {
		fmt.Printf("  [%d] %s %v (seq=%d)\n", i, c.Name, c.Args, c.Seq)
	}
	fmt.Println("=== Read Calls ===")
	for i, c := range m.readCalls {
		fmt.Printf("  [%d] %s (seq=%d)\n", i, c.Path, c.Seq)
	}
	fmt.Println("=== Write Calls ===")
	for i, c := range m.writeCalls {
		fmt.Printf("  [%d] %s (seq=%d)\n", i, c.Path, c.Seq)
	}
}
