package rules

import (
	"fmt"

	"github.com/klyr/klyr/internal/normalize"
)

const defaultDecodeDepth = 2

// Engine evaluates rules against an evaluation context.
type Engine struct {
	Rules []Rule
}

func (e *Engine) Evaluate(ctx EvalContext) Result {
	result := Result{}

	for _, rule := range e.Rules {
		input, ok := selectPhaseInput(ctx, rule.Phase)
		if !ok {
			continue
		}

		normalized, err := applyTransforms(input, rule.Transforms)
		if err != nil {
			continue
		}

		matched, evidence := rule.Matcher.Match(normalized)
		if !matched {
			continue
		}

		result.Score += rule.Score
		result.Matches = append(result.Matches, Match{
			RuleID:   rule.ID,
			Phase:    rule.Phase,
			Score:    rule.Score,
			Tags:     append([]string(nil), rule.Tags...),
			Evidence: evidence,
		})
	}

	return result
}

func selectPhaseInput(ctx EvalContext, phase Phase) (string, bool) {
	switch phase {
	case PhaseRequestLine:
		return ctx.RequestLine.Raw, true
	case PhaseHeaders:
		return ctx.Headers.Raw, true
	case PhaseQuery:
		return ctx.Query.Raw, true
	case PhaseBody:
		return ctx.Body.Raw, true
	default:
		return "", false
	}
}

func applyTransforms(input string, transforms []Transform) (string, error) {
	opts := normalize.Options{MaxDecodeDepth: defaultDecodeDepth}
	for _, transform := range transforms {
		switch transform {
		case TransformLowercase:
			opts.Lowercase = true
		case TransformHTMLEntity:
			opts.HTMLEntity = true
		case TransformPathNormalize:
			opts.NormalizePath = true
		default:
			return "", fmt.Errorf("unknown transform %q", transform)
		}
	}

	res := normalize.Apply(input, opts)
	return res.Normalized, nil
}
