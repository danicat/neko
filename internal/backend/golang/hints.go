package golang

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	undefinedPkgRe = regexp.MustCompile(`undefined:\s+([a-zA-Z0-9_]+)\.`)
	importErrorRe  = regexp.MustCompile(`(?:could not import|package)\s+([a-zA-Z0-9_./-]+)`)
)

// GetDocHintFromOutput checks a raw output string for API usage issues and returns a generic doc hint.
func GetDocHintFromOutput(output string) string {
	return generateHint(output)
}

func generateHint(msg string) string {
	if matches := undefinedPkgRe.FindStringSubmatch(msg); len(matches) > 1 {
		pkgName := matches[1]
		return fmt.Sprintf("\n\n**HINT:** usage of '%s' failed. Try calling `read_docs` on that package to see the correct API.", pkgName)
	}
	if matches := importErrorRe.FindStringSubmatch(msg); len(matches) > 1 {
		pkgPath := matches[1]
		return fmt.Sprintf("\n\n**HINT:** import '%s' failed. Try calling `read_docs` on \"%s\" to verify the package path and exports.", pkgPath, pkgPath)
	}
	return ""
}

// CleanError strips noisy artifacts from Go compiler errors.
func CleanError(msg string) string {
	msg = strings.ReplaceAll(msg, `(invalid package name: "")`, `(invalid package name)`)
	msg = strings.ReplaceAll(msg, `invalid package name: ""`, `invalid package name`)
	return msg
}
