package rules

import "errors"

type AhoMatcher struct {
	nodes []ahoNode
}

type ahoNode struct {
	next map[byte]int
	fail int
	out  []string
}

func NewAhoMatcher(patterns []string) (*AhoMatcher, error) {
	if len(patterns) == 0 {
		return nil, errors.New("patterns are required")
	}

	nodes := []ahoNode{{next: map[byte]int{}, fail: 0}}
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		current := 0
		for i := 0; i < len(pattern); i++ {
			b := pattern[i]
			next, ok := nodes[current].next[b]
			if !ok {
				nodes = append(nodes, ahoNode{next: map[byte]int{}, fail: 0})
				next = len(nodes) - 1
				nodes[current].next[b] = next
			}
			current = next
		}
		nodes[current].out = append(nodes[current].out, pattern)
	}

	queue := make([]int, 0)
	for _, next := range nodes[0].next {
		nodes[next].fail = 0
		queue = append(queue, next)
	}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		for b, next := range nodes[state].next {
			fail := nodes[state].fail
			for fail != 0 {
				if target, ok := nodes[fail].next[b]; ok {
					fail = target
					break
				}
				fail = nodes[fail].fail
			}
			if target, ok := nodes[fail].next[b]; ok {
				nodes[next].fail = target
			} else {
				nodes[next].fail = 0
			}
			nodes[next].out = append(nodes[next].out, nodes[nodes[next].fail].out...)
			queue = append(queue, next)
		}
	}

	if len(nodes) == 1 {
		return nil, errors.New("no non-empty patterns")
	}

	return &AhoMatcher{nodes: nodes}, nil
}

func (m *AhoMatcher) Match(input string) (bool, string) {
	state := 0
	for i := 0; i < len(input); i++ {
		b := input[i]
		for state != 0 {
			if next, ok := m.nodes[state].next[b]; ok {
				state = next
				break
			}
			state = m.nodes[state].fail
		}

		if next, ok := m.nodes[state].next[b]; ok {
			state = next
		}

		if len(m.nodes[state].out) > 0 {
			pattern := m.nodes[state].out[0]
			return true, snippet(pattern)
		}
	}

	return false, ""
}
