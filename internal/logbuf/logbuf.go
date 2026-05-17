package logbuf

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level controls which messages are logged (0=Info, 1=Debug).
type Level int

const (
	LevelInfo   Level = 0
	LevelDebug  Level = 1
	BufSize           = 2000
	maxFileSize       = 5 * 1024 * 1024 // 5 MB
)

// Entry is one log line stored in the ring buffer.
type Entry struct {
	T     time.Time `json:"t"`
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
}

// Logger is a ring-buffer backed logger with optional file output.
type Logger struct {
	mu        sync.RWMutex
	buf       [BufSize]Entry
	head      int // next write index (mod BufSize)
	count     int // number of valid entries (capped at BufSize)
	level     Level
	file      *os.File
	filePath  string
	fileBytes int64
}

// Global is the process-wide logger. Wire it with log.SetOutput(Global.Writer()).
var Global = &Logger{}

func (l *Logger) SetLevel(lv Level) {
	l.mu.Lock()
	l.level = lv
	l.mu.Unlock()
}

func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetFile opens (or creates/appends) a log file. Pass "" to disable file output.
func (l *Logger) SetFile(path string) error {
	if path == "" {
		l.mu.Lock()
		old := l.file
		l.file = nil
		l.filePath = ""
		l.fileBytes = 0
		l.mu.Unlock()
		if old != nil {
			old.Close()
		}
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	var size int64
	if info, err := f.Stat(); err == nil {
		size = info.Size()
	}
	l.mu.Lock()
	old := l.file
	l.file = f
	l.filePath = path
	l.fileBytes = size
	l.mu.Unlock()
	if old != nil {
		old.Close()
	}
	return nil
}

// rotate renames the current log file to .bak and opens a fresh one.
// Must be called with l.mu held.
func (l *Logger) rotate() {
	if l.file == nil {
		return
	}
	l.file.Close()
	_ = os.Rename(l.filePath, l.filePath+".bak")
	f, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		l.file = nil
		return
	}
	l.file = f
	l.fileBytes = 0
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.write(LevelInfo, "INFO", fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.write(LevelDebug, "DEBUG", fmt.Sprintf(format, args...))
}

func (l *Logger) write(msgLevel Level, levelStr, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if msgLevel > l.level {
		return
	}
	e := Entry{T: time.Now(), Level: levelStr, Msg: msg}
	l.buf[l.head%BufSize] = e
	l.head++
	if l.count < BufSize {
		l.count++
	}
	line := e.T.Format("2006-01-02 15:04:05") + " [" + levelStr + "] " + msg + "\n"
	if l.file != nil {
		if l.fileBytes >= maxFileSize {
			l.rotate()
		}
		n, _ := l.file.WriteString(line)
		l.fileBytes += int64(n)
	}
	_, _ = os.Stderr.WriteString(line)
}

// Entries returns all buffered entries in chronological order (oldest first).
func (l *Logger) Entries() []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.count == 0 {
		return nil
	}
	result := make([]Entry, l.count)
	start := 0
	if l.count == BufSize {
		start = l.head % BufSize
	}
	for i := 0; i < l.count; i++ {
		result[i] = l.buf[(start+i)%BufSize]
	}
	return result
}

// Writer returns an io.Writer that routes writes into this logger at INFO level.
// Use log.SetOutput(logbuf.Global.Writer()) to capture the standard log package.
func (l *Logger) Writer() io.Writer {
	return &bridgeWriter{l: l}
}

type bridgeWriter struct{ l *Logger }

func (w *bridgeWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n\r")
	if msg != "" {
		w.l.Info("%s", msg)
	}
	return len(p), nil
}
