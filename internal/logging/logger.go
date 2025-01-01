package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level   Level
	logFile *os.File
	logger  *log.Logger
	verbose bool
}

func NewLogger(logFile string, level Level, verbose bool) (*Logger, error) {
	var file *os.File
	var err error

	if logFile != "" {
		// Create log directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		file, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	l := &Logger{
		level:   level,
		logFile: file,
		logger:  log.New(os.Stdout, "", 0),
		verbose: verbose,
	}

	if file != nil {
		l.logger.SetOutput(file)
	}

	return l, nil
}

func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Format level string
	levelStr := "UNKNOWN"
	switch level {
	case DEBUG:
		levelStr = "DEBUG"
	case INFO:
		levelStr = "INFO"
	case WARN:
		levelStr = "WARN"
	case ERROR:
		levelStr = "ERROR"
	}

	// Format message
	msg := fmt.Sprintf(format, args...)

	// Format log entry
	entry := fmt.Sprintf("%s [%s] %s:%d: %s",
		timestamp,
		levelStr,
		filepath.Base(file),
		line,
		msg)

	// Write to log
	l.logger.Println(entry)

	// Print to stdout if verbose mode is enabled
	if l.verbose {
		// Use different colors for different levels
		var colorCode string
		switch level {
		case DEBUG:
			colorCode = "\033[36m" // Cyan
		case INFO:
			colorCode = "\033[32m" // Green
		case WARN:
			colorCode = "\033[33m" // Yellow
		case ERROR:
			colorCode = "\033[31m" // Red
		}

		fmt.Printf("%s%s\033[0m\n", colorCode, entry)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Progress logging for longer operations
type ProgressLogger struct {
	logger  *Logger
	total   int
	current int
	message string
}

func (l *Logger) NewProgress(total int, message string) *ProgressLogger {
	return &ProgressLogger{
		logger:  l,
		total:   total,
		current: 0,
		message: message,
	}
}

func (p *ProgressLogger) Increment() {
	p.current++
	p.logProgress()
}

func (p *ProgressLogger) SetCurrent(current int) {
	p.current = current
	p.logProgress()
}

func (p *ProgressLogger) logProgress() {
	percentage := float64(p.current) / float64(p.total) * 100
	progressBar := p.createProgressBar(percentage)

	p.logger.Info("%s: %s %.1f%% (%d/%d)",
		p.message,
		progressBar,
		percentage,
		p.current,
		p.total)
}

func (p *ProgressLogger) createProgressBar(percentage float64) string {
	width := 30
	completed := int(float64(width) * percentage / 100)

	var bar strings.Builder
	bar.WriteString("[")
	for i := 0; i < width; i++ {
		if i < completed {
			bar.WriteString("=")
		} else if i == completed {
			bar.WriteString(">")
		} else {
			bar.WriteString(" ")
		}
	}
	bar.WriteString("]")

	return bar.String()
}
