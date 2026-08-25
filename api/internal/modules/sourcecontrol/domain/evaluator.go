package domain

import (
	"path"
	"regexp"
	"strings"
)

func EvaluateTrigger(rules BuildTriggerRules, changedFiles []string, filesKnown bool) TriggerDecision {
	if rules.Branch == "" {
		rules.Branch = "main"
	}
	if !rules.AutoDeploy {
		return TriggerDecision{Reason: ReasonAutodeployDisabled}
	}
	if !filesKnown {
		return TriggerDecision{Trigger: true, Reason: ReasonChangedFilesUnknown}
	}
	if len(changedFiles) == 0 {
		return TriggerDecision{Reason: ReasonNoRelevantChanges}
	}

	ignore := compilePatterns(rules.IgnorePaths)
	watch := compilePatterns(rules.WatchPaths)
	matches := make([]string, 0)
	for _, file := range changedFiles {
		file = normalizePath(file)
		if file == "" || matchesAny(ignore, file) {
			continue
		}
		if len(watch) == 0 && rules.RootDirectory == "" {
			matches = append(matches, file)
			continue
		}
		if rules.WatchRootFiles && !strings.Contains(file, "/") {
			matches = append(matches, file)
			continue
		}
		if rules.RootDirectory != "" && pathWithin(rules.RootDirectory, file) {
			matches = append(matches, file)
			continue
		}
		if matchesAny(watch, file) {
			matches = append(matches, file)
		}
	}
	if len(matches) == 0 {
		return TriggerDecision{Reason: ReasonNoRelevantChanges}
	}
	reason := ReasonWatchPathMatched
	if rules.WatchRootFiles {
		for _, match := range matches {
			if !strings.Contains(match, "/") {
				reason = ReasonRootFileMatched
				break
			}
		}
	}
	return TriggerDecision{Trigger: true, Reason: reason, Matches: matches}
}

type compiledPattern struct {
	RootOnly bool
	Regex    *regexp.Regexp
}

func compilePatterns(patterns []string) []compiledPattern {
	compiled := make([]compiledPattern, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || strings.HasPrefix(pattern, "!") {
			continue
		}
		rootOnly := !strings.Contains(pattern, "/")
		regex, err := globRegex(normalizePath(pattern))
		if err == nil {
			compiled = append(compiled, compiledPattern{RootOnly: rootOnly, Regex: regex})
		}
	}
	return compiled
}

func matchesAny(patterns []compiledPattern, file string) bool {
	for _, pattern := range patterns {
		if pattern.RootOnly && strings.Contains(file, "/") {
			continue
		}
		if pattern.Regex.MatchString(file) {
			return true
		}
	}
	return false
}

func globRegex(pattern string) (*regexp.Regexp, error) {
	var out strings.Builder
	out.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					out.WriteString("(?:.*/)?")
				} else {
					out.WriteString(".*")
				}
			} else {
				out.WriteString("[^/]*")
			}
		case '?':
			out.WriteString("[^/]")
		default:
			out.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	out.WriteString("$")
	return regexp.Compile(out.String())
}

func pathWithin(root, file string) bool {
	root = strings.Trim(normalizePath(root), "/")
	return root != "" && (file == root || strings.HasPrefix(file, root+"/"))
}

func normalizePath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	return strings.TrimPrefix(path.Clean("/"+value), "/")
}
