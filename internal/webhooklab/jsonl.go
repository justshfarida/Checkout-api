package webhooklab

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONLFile appends one JSON object per line (JSON Lines, ".jsonl").
// Used for the event log (task 1) and the dead-letter file (task 8).
// Appending a line is cheap, a crash can only lose the last line, and the
// file is easy to read back line by line (task 6: replay).
type JSONLFile struct {
	mu   sync.Mutex // several goroutines write at the same time
	path string
	f    *os.File
}

// OpenJSONL opens (or creates) the file in append mode.
func OpenJSONL(path string) (*JSONLFile, error) {
	path = filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create dir for %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return &JSONLFile{path: path, f: f}, nil
}

// Append writes v as one line. json.Marshal never emits a raw newline, so
// one value is always exactly one line.
func (j *JSONLFile) Append(v any) error {
	line, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode line: %w", err)
	}
	line = append(line, '\n')

	j.mu.Lock()
	defer j.mu.Unlock()
	if _, err := j.f.Write(line); err != nil {
		return fmt.Errorf("write %s: %w", j.path, err)
	}
	return nil
}

// Path is the file's location, e.g. for ReadEventLog.
func (j *JSONLFile) Path() string { return j.path }

// Close closes the file.
func (j *JSONLFile) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.f.Close()
}

// LogEntry is one line of the event log (task 1).
type LogEntry struct {
	ReceivedAt time.Time       `json:"received_at"`
	EventID    string          `json:"event_id"`
	Event      string          `json:"event"`
	Payload    json.RawMessage `json:"payload"` // the original body, for replay
}

// DeadLetter is one line of dead_letter.jsonl (task 8).
type DeadLetter struct {
	FailedAt time.Time `json:"failed_at"`
	Handler  string    `json:"handler"`
	Attempts int       `json:"attempts"`
	Error    string    `json:"error"`
	Event    Event     `json:"event"`
}

// ReadEventLog reads all entries of an event log file, oldest first.
func ReadEventLog(path string) ([]LogEntry, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*maxBodyBytes) // a line holds a whole body
	line := 0
	for scanner.Scan() {
		line++
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var e LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, line, err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return entries, nil
}
