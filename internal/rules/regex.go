package rules

import "regexp"

type RegexMatcher struct {
	re *regexp.Regexp
}

func NewRegexMatcher(pattern string) (*RegexMatcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexMatcher{re: re}, nil
}

func (m *RegexMatcher) Match(input string) (bool, string) {
	loc := m.re.FindStringIndex(input)
	if loc == nil {
		return false, ""
	}
	return true, snippet(input[loc[0]:loc[1]])
}
