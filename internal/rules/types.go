package rules

type Phase string

type MatchType string

type Transform string

const (
	PhaseRequestLine Phase = "request_line"
	PhaseHeaders     Phase = "headers"
	PhaseQuery       Phase = "query"
	PhaseBody        Phase = "body"
)

const (
	MatchRegex MatchType = "regex"
	MatchAho   MatchType = "aho"
)

const (
	TransformLowercase     Transform = "lowercase"
	TransformHTMLEntity    Transform = "html_entity"
	TransformPathNormalize Transform = "normalize_path"
)

type Rule struct {
	ID         string
	Phase      Phase
	Score      int
	Tags       []string
	Transforms []Transform
	Matcher    Matcher
}

type Match struct {
	RuleID   string
	Phase    Phase
	Score    int
	Tags     []string
	Evidence string
}

type Result struct {
	Score   int
	Matches []Match
}

// Matcher returns true if the input matches and an optional evidence snippet.
// Evidence must be a small redacted snippet (max 64 chars) in later stages.
type Matcher interface {
	Match(input string) (bool, string)
}
