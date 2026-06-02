package setup

import (
	"strconv"
	"strings"
	"testing"
)

// swiftBarInterval reimplements SwiftBar's plugin-filename interval parser
// (see SwiftBar/Plugin/ExecutablePlugin.swift and
// SwiftBar/Utility/PluginUtilities.swift in github.com/swiftbar/SwiftBar).
//
// SwiftBar splits the filename on "." and only considers interval tokens
// when the split produces more than two components. For each middle
// component it strips non-digits, parses the result as a number, and
// applies a unit multiplier from the trailing letter: s/m/h/d. The final
// interval is the minimum of every interval token it could parse.
//
// Returns (interval in seconds, ok). ok is false when no interval was
// recognised — SwiftBar's behaviour in that case is to fall back to its
// 100-day "never refresh" default, which is the bug this regression test
// guards against.
func swiftBarInterval(filename string) (float64, bool) {
	parts := strings.Split(filename, ".")
	if len(parts) <= 2 {
		return 0, false
	}
	best := -1.0
	for _, p := range parts[1:] {
		digits := strings.Map(func(r rune) rune {
			if (r >= '0' && r <= '9') || r == '.' {
				return r
			}
			return -1
		}, p)
		if digits == "" {
			continue
		}
		n, err := strconv.ParseFloat(digits, 64)
		if err != nil {
			continue
		}
		mult := 0.0
		switch {
		case strings.HasSuffix(p, "s"):
			mult = 1
		case strings.HasSuffix(p, "m"):
			mult = 60
		case strings.HasSuffix(p, "h"):
			mult = 3600
		case strings.HasSuffix(p, "d"):
			mult = 86400
		default:
			continue
		}
		secs := n * mult
		if best < 0 || secs < best {
			best = secs
		}
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}

func TestPluginFilenameIsSwiftBarParseable(t *testing.T) {
	got, ok := swiftBarInterval(PluginFilename)
	if !ok {
		t.Fatalf("SwiftBar would not recognise an interval in %q — "+
			"the filename needs at least 3 dot-separated components "+
			"(name.<interval>.<ext>). Without this SwiftBar falls back "+
			"to its 100-day default and the plugin never refreshes.",
			PluginFilename)
	}
	const want = 30.0
	if got != want {
		t.Errorf("SwiftBar would parse %q as %vs, want %vs",
			PluginFilename, got, want)
	}
}

func TestSwiftBarIntervalParser(t *testing.T) {
	// These cases document SwiftBar's filename rules so future changes
	// to PluginFilename can be reasoned about without re-deriving the
	// parser from the Swift source.
	cases := []struct {
		name     string
		filename string
		wantSecs float64
		wantOK   bool
	}{
		{"no interval, no extension", "gmailnotifier", 0, false},
		{"interval but no trailing extension (the v2.0.0 bug)", "gmailnotifier.30s", 0, false},
		{"interval with bin extension", "gmailnotifier.30s.bin", 30, true},
		{"interval with sh extension", "gmailnotifier.30s.sh", 30, true},
		{"minutes", "x.5m.sh", 300, true},
		{"hours", "x.2h.sh", 7200, true},
		{"days", "x.1d.sh", 86400, true},
		{"multiple intervals, smallest wins", "x.5m.30s.sh", 30, true},
		{"unrecognised unit falls through", "x.30x.sh", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := swiftBarInterval(tc.filename)
			if ok != tc.wantOK {
				t.Fatalf("swiftBarInterval(%q) ok=%v, want %v", tc.filename, ok, tc.wantOK)
			}
			if ok && got != tc.wantSecs {
				t.Errorf("swiftBarInterval(%q) = %v, want %v", tc.filename, got, tc.wantSecs)
			}
		})
	}
}
