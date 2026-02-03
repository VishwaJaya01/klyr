package rules

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klyr/klyr/internal/config"
)

func BuildEngine(cfg *config.Config, baseDir string) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	rules := make([]Rule, 0, len(cfg.Rules))
	for _, raw := range cfg.Rules {
		compiled, err := compileRule(raw, baseDir)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", raw.ID, err)
		}
		rules = append(rules, compiled)
	}

	return &Engine{Rules: rules}, nil
}

func compileRule(raw config.Rule, baseDir string) (Rule, error) {
	phase := Phase(raw.Phase)
	matchType := MatchType(raw.Match.Type)

	transforms, err := mapTransforms(raw.Transforms)
	if err != nil {
		return Rule{}, err
	}

	var matcher Matcher
	switch matchType {
	case MatchRegex:
		if raw.Match.Pattern == "" {
			return Rule{}, fmt.Errorf("regex pattern is required")
		}
		matcher, err = NewRegexMatcher(raw.Match.Pattern)
	case MatchAho:
		if raw.Match.PatternsFile == "" {
			return Rule{}, fmt.Errorf("patternsFile is required")
		}
		patterns, readErr := readPatterns(resolvePath(baseDir, raw.Match.PatternsFile))
		if readErr != nil {
			return Rule{}, readErr
		}
		patterns = applyPatternTransforms(patterns, transforms)
		matcher, err = NewAhoMatcher(patterns)
	default:
		return Rule{}, fmt.Errorf("unknown match type %q", matchType)
	}
	if err != nil {
		return Rule{}, err
	}

	return Rule{
		ID:         raw.ID,
		Phase:      phase,
		Score:      raw.Score,
		Tags:       append([]string(nil), raw.Tags...),
		Transforms: transforms,
		Matcher:    matcher,
	}, nil
}

func mapTransforms(raw []string) ([]Transform, error) {
	out := make([]Transform, 0, len(raw))
	for _, item := range raw {
		transform := Transform(strings.TrimSpace(item))
		switch transform {
		case TransformLowercase, TransformHTMLEntity, TransformPathNormalize:
			out = append(out, transform)
		default:
			return nil, fmt.Errorf("unknown transform %q", item)
		}
	}
	return out, nil
}

func applyPatternTransforms(patterns []string, transforms []Transform) []string {
	lower := false
	for _, t := range transforms {
		if t == TransformLowercase {
			lower = true
			break
		}
	}
	if !lower {
		return patterns
	}

	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, strings.ToLower(p))
	}
	return out
}

func readPatterns(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}

func resolvePath(baseDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if baseDir == "" {
		return path
	}
	return filepath.Join(baseDir, path)
}
