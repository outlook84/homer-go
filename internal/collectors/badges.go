package collectors

import "fmt"

func countBadge(label string, value int, state string, tone string) Badge {
	return Badge{
		Label: label,
		Value: fmt.Sprintf("%d", value),
		State: state,
		Tone:  tone,
	}
}

func positiveBadges(badges ...Badge) []Badge {
	out := make([]Badge, 0, len(badges))
	for _, badge := range badges {
		if badge.Value != "0" {
			out = append(out, badge)
		}
	}
	return out
}
