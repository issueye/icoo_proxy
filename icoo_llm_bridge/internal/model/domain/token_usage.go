package domain

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

func (u TokenUsage) Normalize() TokenUsage {
	if u.TotalTokens <= 0 {
		u.TotalTokens = u.InputTokens + u.OutputTokens
	}
	return u
}
