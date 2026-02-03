package normalize

import "strings"

func NormalizePath(path string) string {
	if path == "" {
		return "/"
	}

	leading := strings.HasPrefix(path, "/")
	trailing := strings.HasSuffix(path, "/") && path != "/"

	parts := strings.Split(path, "/")
	stack := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
			stack = append(stack, part)
		}
	}

	var b strings.Builder
	if leading {
		b.WriteString("/")
	}
	b.WriteString(strings.Join(stack, "/"))
	if trailing && b.Len() > 1 {
		b.WriteString("/")
	}

	out := b.String()
	if out == "" {
		return "/"
	}
	return out
}
