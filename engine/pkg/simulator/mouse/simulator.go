package mouse

import "math"

// ScalePath rescales recorded mouse actions to a new width (Java MouseEventSimulator subset).
func ScalePath(actions []Action, targetWidth int) []Action {
	if len(actions) == 0 || targetWidth <= 0 {
		return actions
	}
	p1, p2 := findPressRelease(actions)
	width := float64(p2[0] - p1[0])
	if width == 0 {
		return actions
	}
	scale := float64(targetWidth) / width
	out := make([]Action, len(actions))
	copy(out, actions)
	for i := range out {
		if out[i].Type == TypeMove || out[i].Type == TypePress || out[i].Type == TypeRelease {
			out[i].X = p1[0] + int(math.Round(float64(out[i].X-p1[0])*scale))
			out[i].Y = p1[1] + int(math.Round(float64(out[i].Y-p1[1])*scale))
		}
	}
	return out
}

func findPressRelease(actions []Action) (p1, p2 [2]int) {
	for _, a := range actions {
		if a.Type == TypePress {
			p1 = [2]int{a.X, a.Y}
		}
		if a.Type == TypeRelease {
			p2 = [2]int{a.X, a.Y}
		}
	}
	return p1, p2
}
