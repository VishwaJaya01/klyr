package rules

const maxEvidence = 64

func snippet(value string) string {
	if len(value) <= maxEvidence {
		return value
	}
	return value[:maxEvidence]
}
