package policy

import "github.com/klyr/klyr/internal/normalize"

type EvalContext struct {
	RequestLine normalize.Result
	Headers     normalize.Result
	Query       normalize.Result
	Body        normalize.Result
}
