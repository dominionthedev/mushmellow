package ci

import "strings"

type Mode string

const (
	SoftMode  Mode = "soft"
	CIMode    Mode = "ci"
	QuietMode Mode = "quiet"
)

func FromString(s string) (Mode, bool) {
	switch strings.ToLower(s) {
	case "soft":
		return SoftMode, true
	case "ci":
		return CIMode, true
	case "quiet":
		return QuietMode, true
	default:
		return SoftMode, false
	}
}
