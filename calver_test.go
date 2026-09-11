package semver

import "testing"

// TestBareDatestampIsDemotedButADottedCalVerIsNot.
//
// pkgx labels a pre-release with a datestamp, so a BARE `20230101` must not
// outrank `1.0.0` — that is what the demotion is for, and it stays.
//
// Keying it on the major alone also caught a DOTTED CalVer. `2026.09.07.00` is
// not a pre-release label; it is how some projects number their releases, and
// demoting it made their newest version a decade old.
func TestBareDatestampIsDemotedButADottedCalVerIsNot(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
		why  string
	}{
		// the rule this demotion exists for, unchanged
		{"20230101", "1.0.0", -1, "a bare datestamp is a pre-release label"},
		{"1.0.0", "20230101", 1, "and the other way round"},
		{"20230101", "20240101", -1, "two datestamps still compare as numbers"},

		// what it was also catching
		{"2026.09.07.00", "0.57.0", 1, "facebook.com/folly: CalVer release line vs its old SemVer tags"},
		{"2026.09.07.00", "0.13.0", 1, "facebook.com/wangle"},
		{"2026.09.07.00", "0.31.0", 1, "facebook.com/fbthrift"},
		{"2026.09.07.00", "1.2.3", 1, "a year is larger than one"},
		{"1.2.3", "2026.09.07.00", -1, "and the other way round"},

		// ordering inside a CalVer line is untouched
		{"2026.09.07.00", "2026.08.24.00", 1, "same length, later date"},
		{"2026.8.24.0", "2026.9.7.0", -1, "unpadded spelling of the same"},

		// and inside a SemVer line
		{"1.2.3", "1.2.4", -1, "plain semver"},
		{"1.2.3", "1.2.3.0", 0, "trailing components default to zero"},
	} {
		a, b := MustParseVersion(tc.a), MustParseVersion(tc.b)
		if got := a.Compare(b); got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d — %s", tc.a, tc.b, got, tc.want, tc.why)
		}
	}
}

// TestDemotionNeedsExactlyOneComponent is the boundary: a two-component CalVer
// is a version, not a datestamp, and must not be demoted.
func TestDemotionNeedsExactlyOneComponent(t *testing.T) {
	if !MustParseVersion("2026.1").GT(MustParseVersion("9.9.9")) {
		t.Error("a two-component CalVer was demoted")
	}
	if !MustParseVersion("20260101").LT(MustParseVersion("0.0.1")) {
		t.Error("a one-component datestamp was not demoted")
	}
}
