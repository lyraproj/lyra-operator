package px

import (
	"github.com/lyraproj/issue/issue"
)

// LogWarning logs a warning to the logger of the current context
func LogWarning(issueCode issue.Code, args issue.H) {
	CurrentContext().Logger().LogIssue(Warning(issueCode, args))
}

// Error creates a Reported with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
func Error(issueCode issue.Code, args issue.H) issue.Reported {
	return issue.NewReported(issueCode, issue.SeverityError, args, 1)
}

// Error2 creates a Reported with the given issue code, location from stack top, and arguments
// Typical use is to panic with the returned value
func Error2(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
	return issue.NewReported(issueCode, issue.SeverityError, args, location)
}

// Warning creates a Reported with the given issue code, location from stack top, and arguments
// and logs it on the currently active logger
func Warning(issueCode issue.Code, args issue.H) issue.Reported {
	c := CurrentContext()
	ri := issue.NewReported(issueCode, issue.SeverityWarning, args, 1)
	c.Logger().LogIssue(ri)
	return ri
}
