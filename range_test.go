package semver

import "testing"

func mustRange(t *testing.T, s string) *Range {
	t.Helper()
	r, err := NewRange(s)
	if err != nil {
		t.Fatalf("NewRange(%q): %v", s, err)
	}
	return r
}

func TestRangeSatisfies(t *testing.T) {
	cases := []struct {
		rng, ver string
		want     bool
	}{
		{"*", "9.9.9", true},
		{"^1", "1.5.0", true},
		{"^1", "2.0.0", false},
		{"^1", "1.0.0", true},
		{"^0.2.3", "0.2.9", true},
		{"^0.2.3", "0.3.0", false},
		{"~1.2", "1.2.9", true},
		{"~1.2", "1.3.0", false},
		{"~1", "1.9.9", true},
		{"~1", "2.0.0", false},
		{"<3", "2.9.9", true},
		{"<3", "3.0.0", false},
		{"=1.2.3", "1.2.3", true},
		{"=1.2.3", "1.2.4", false},
		{"@5", "5.9.9", true},
		{"@5", "6.0.0", false},
		{"@5.1", "5.1.9", true},
		{"@5.1", "5.2.0", false},
		{">=1.2.0 <2.0.0", "1.5.0", true},
		{">=1.2.0 <2.0.0", "2.0.0", false},
		{">=2", "9.0.0", true},
		{">=2", "1.0.0", false},
		{"^1||^3", "3.1.0", true},
		{"^1,^3", "2.0.0", false},
	}
	for _, c := range cases {
		if got := mustRange(t, c.rng).Satisfies(MustParseVersion(c.ver)); got != c.want {
			t.Errorf("%q.Satisfies(%q) = %v, want %v", c.rng, c.ver, got, c.want)
		}
	}
}

func TestRangeParseTolerant(t *testing.T) {
	// a bare version is treated as @version
	r, err := ParseRange("5.1")
	if err != nil || !r.Satisfies(MustParseVersion("5.1.9")) || r.Satisfies(MustParseVersion("5.2.0")) {
		t.Errorf("tolerant bare = %v %v", r, err)
	}
	if _, err := ParseRange(""); err == nil {
		t.Error("empty range should error")
	}
	if _, err := ParseRange("!!garbage"); err == nil {
		t.Error("garbage range should error")
	}
	// a valid operator range goes through NewRange directly
	if r, err := ParseRange("^2"); err != nil || !r.Satisfies(MustParseVersion("2.3.0")) {
		t.Errorf("operator via ParseRange = %v %v", r, err)
	}
}

func TestRangeErrors(t *testing.T) {
	for _, bad := range []string{"", "garbage", ">=2 <1", "^", "@x", ">=x", "^x", "~x", "<x", "=x"} {
		if _, err := NewRange(bad); err == nil {
			t.Errorf("NewRange(%q) should error", bad)
		}
	}
}

func TestRangeStringRoundTrip(t *testing.T) {
	cases := map[string]string{
		"*":              "*",
		"^1":             "^1",
		"^1.2.3":         "^1.2.3",
		"~1.2":           "~1.2",
		">=2":            ">=2",
		"=1.2.3":         "=1.2.3",
		">=1.2.0 <2.0.5": ">=1.2<2.0.5",
		"@5":             "^5",        // @5 and ^5 denote the same [5,6)
		"@5.1":           "~5.1",      // @5.1 == ~5.1
		"@5.1.0":         "@5.1.0",    // the genuine @-form
		"^0":             "^0",        // [0,1), single component
		"^0.1.2":         "~0.1.2",    // [0.1.2,0.2.0) renders as tilde
		">=0.1.2 <1":     ">=0.1.2<1", // the 0.x-to-1 caret-zero form
		// general >=x<y forms that exercise atBound's non-@ paths:
		">=1.0.4 <2.0.5":   ">=1.0.4<2.0.5",   // prefix differs
		">=1.2 <1.2.5":     ">=1.2<1.2.5",     // v1 shorter than v2 (pad)
		">=1.2.3.4 <1.2.4": ">=1.2.3.4<1.2.4", // v1 longer than v2
	}
	for in, want := range cases {
		if got := mustRange(t, in).String(); got != want {
			t.Errorf("NewRange(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestRangeMaxAndSingle(t *testing.T) {
	r := mustRange(t, "^1")
	vs := []*Version{MustParseVersion("1.0.0"), MustParseVersion("1.9.0"), MustParseVersion("2.0.0")}
	if m := r.Max(vs); m == nil || m.String() != "1.9.0" {
		t.Errorf("Max = %v", m)
	}
	if m := mustRange(t, "^9").Max(vs); m != nil {
		t.Errorf("Max no-match = %v", m)
	}
	if s := mustRange(t, "=1.2.3").Single(); s == nil || s.String() != "1.2.3" {
		t.Errorf("Single = %v", s)
	}
	if mustRange(t, "*").Single() != nil || mustRange(t, "^1").Single() != nil {
		t.Error("star/interval Single should be nil")
	}
}

func TestIntersect(t *testing.T) {
	star := mustRange(t, "*")
	c1 := mustRange(t, "^1")
	if got, _ := Intersect(c1, star); got != c1 {
		t.Error("intersect with * (right)")
	}
	if got, _ := Intersect(star, c1); got != c1 {
		t.Error("intersect with * (left)")
	}
	// single ∩ single
	if got, err := Intersect(mustRange(t, "=1.2.3"), mustRange(t, "=1.2.3")); err != nil || !got.Satisfies(MustParseVersion("1.2.3")) {
		t.Errorf("single∩single = %v %v", got, err)
	}
	// single ∩ interval (both orders)
	if got, err := Intersect(mustRange(t, "=1.5.0"), c1); err != nil || !got.Satisfies(MustParseVersion("1.5.0")) {
		t.Errorf("single∩interval = %v %v", got, err)
	}
	if got, err := Intersect(c1, mustRange(t, "=1.5.0")); err != nil || !got.Satisfies(MustParseVersion("1.5.0")) {
		t.Errorf("interval∩single = %v %v", got, err)
	}
	// interval ∩ interval → overlap [1.2,2)
	got, err := Intersect(c1, mustRange(t, "^1.2"))
	if err != nil || !got.Satisfies(MustParseVersion("1.5.0")) || got.Satisfies(MustParseVersion("1.1.0")) {
		t.Errorf("interval∩interval = %v %v", got, err)
	}
	// interval ∩ interval where BOTH bounds tighten from the other side
	if got, err := Intersect(mustRange(t, "~1.2"), c1); err != nil ||
		!got.Satisfies(MustParseVersion("1.2.5")) || got.Satisfies(MustParseVersion("1.3.0")) {
		t.Errorf("tightened interval = %v %v", got, err)
	}
	// disjoint → error
	if _, err := Intersect(mustRange(t, "^1"), mustRange(t, "^3")); err == nil {
		t.Error("disjoint intersect should error")
	}
	// single not in interval → empty → error
	if _, err := Intersect(mustRange(t, "=9.0.0"), c1); err == nil {
		t.Error("single-out-of-interval should error")
	}
}
