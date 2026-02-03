package policy

import "github.com/klyr/klyr/internal/rules"

type Action string

const (
	ActionAllow  Action = "allow"
	ActionBlock  Action = "block"
	ActionShadow Action = "shadow"
)

func EvaluateRules(engine *rules.Engine, ctx EvalContext) rules.Result {
	if engine == nil {
		return rules.Result{}
	}
	return engine.Evaluate(ctx)
}

func DecideAction(mode string, score, threshold int) (Action, bool) {
	if score < threshold {
		return ActionAllow, false
	}

	switch mode {
	case "shadow":
		return ActionShadow, false
	case "enforce":
		return ActionBlock, true
	case "learn":
		return ActionAllow, false
	default:
		return ActionAllow, false
	}
}
