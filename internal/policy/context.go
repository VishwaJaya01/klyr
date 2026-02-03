package policy

type Field struct {
	Raw        string
	Normalized string
}

type EvalContext struct {
	RequestLine Field
	Headers     Field
	Query       Field
	Body        Field
}
