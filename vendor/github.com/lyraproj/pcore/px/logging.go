package px

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/lyraproj/pcore/utils"

	"github.com/lyraproj/issue/issue"
)

type (
	LogLevel string

	Logger interface {
		Log(level LogLevel, args ...Value)

		Logf(level LogLevel, format string, args ...interface{})

		LogIssue(issue issue.Reported)
	}

	stdlog struct {
	}

	LogEntry interface {
		Level() LogLevel
		Message() string
	}

	ArrayLogger struct {
		entries []LogEntry
	}

	ReportedEntry struct {
		issue issue.Reported
	}

	TextEntry struct {
		level   LogLevel
		message string
	}
)

const (
	ALERT   = LogLevel(`alert`)
	CRIT    = LogLevel(`crit`)
	DEBUG   = LogLevel(`debug`)
	EMERG   = LogLevel(`emerg`)
	ERR     = LogLevel(`err`)
	INFO    = LogLevel(`info`)
	NOTICE  = LogLevel(`notice`)
	WARNING = LogLevel(`warning`)
	IGNORE  = LogLevel(``)
)

var LogLevels = []LogLevel{ALERT, CRIT, DEBUG, EMERG, ERR, INFO, NOTICE, WARNING}

func (l LogLevel) Severity() issue.Severity {
	switch l {
	case CRIT, EMERG, ERR:
		return issue.SeverityError
	case ALERT, WARNING:
		return issue.SeverityWarning
	default:
		return issue.SeverityIgnore
	}
}

func NewStdLogger() Logger {
	return &stdlog{}
}

func (l *stdlog) Log(level LogLevel, args ...Value) {
	w := l.writerFor(level)
	utils.Fprintf(w, `%s: `, level)
	for _, arg := range args {
		ToString3(arg, w)
	}
	utils.Fprintln(w)
}

func (l *stdlog) Logf(level LogLevel, format string, args ...interface{}) {
	w := l.writerFor(level)
	utils.Fprintf(w, `%s: `, level)
	utils.Fprintf(w, format, args...)
	utils.Fprintln(w)
}

func (l *stdlog) writerFor(level LogLevel) io.Writer {
	switch level {
	case DEBUG, INFO, NOTICE:
		return os.Stdout
	default:
		return os.Stderr
	}
}

func (l *stdlog) LogIssue(issue issue.Reported) {
	utils.Fprintln(os.Stderr, issue.String())
}

func NewArrayLogger() *ArrayLogger {
	return &ArrayLogger{make([]LogEntry, 0, 16)}
}

func (l *ArrayLogger) Entries(level LogLevel) []LogEntry {
	result := make([]LogEntry, 0, 8)
	for _, entry := range l.entries {
		if entry.Level() == level {
			result = append(result, entry)
		}
	}
	return result
}

func (l *ArrayLogger) Log(level LogLevel, args ...Value) {
	w := bytes.NewBufferString(``)
	for _, arg := range args {
		ToString3(arg, w)
	}
	l.entries = append(l.entries, &TextEntry{level, w.String()})
}

func (l *ArrayLogger) Logf(level LogLevel, format string, args ...interface{}) {
	l.entries = append(l.entries, &TextEntry{level, fmt.Sprintf(format, args...)})
}

func (l *ArrayLogger) LogIssue(i issue.Reported) {
	if i.Severity() != issue.SeverityIgnore {
		l.entries = append(l.entries, &ReportedEntry{i})
	}
}

func (te *TextEntry) Level() LogLevel {
	return te.level
}

func (te *TextEntry) Message() string {
	return te.message
}

func (re *ReportedEntry) Level() LogLevel {
	return LogLevelFromSeverity(re.issue.Severity())
}

func (re *ReportedEntry) Message() string {
	return re.issue.String()
}

func (re *ReportedEntry) Issue() issue.Reported {
	return re.issue
}

func LogLevelFromSeverity(severity issue.Severity) LogLevel {
	switch severity {
	case issue.SeverityError:
		return ERR
	case issue.SeverityWarning, issue.SeverityDeprecation:
		return WARNING
	default:
		return IGNORE
	}
}
