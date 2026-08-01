package semver

import "testing"

func TestParseVersionForms(t *testing.T) {
	v, err := ParseVersion("v1.2.3")
	if err != nil || v.Major != 1 || v.Minor != 2 || v.Patch != 3 || v.String() != "1.2.3" {
		t.Fatalf("plain = %+v %v", v, err)
	}
	// openssl-style trailing letter: w = 23rd letter
	w := MustParseVersion("1.1.1w")
	if len(w.Components) != 4 || w.Components[3] != 23 || w.String() != "1.1.1w" {
		t.Errorf("lettered = %v str=%q", w.Components, w.String())
	}
	// four components (ghc 5.64.3.2)
	g := MustParseVersion("5.64.3.2")
	if g.String() != "5.64.3.2" || g.Patch != 3 {
		t.Errorf("4-comp = %q", g.String())
	}
	// single component pads minor/patch to 0
	if s := MustParseVersion("7"); s.Minor != 0 || s.Patch != 0 || s.String() != "7.0.0" {
		t.Errorf("single = %q", s.String())
	}
}

func TestParseVersionErrors(t *testing.T) {
	for _, bad := range []string{"", "abc", "1.x", "1w.2", "1.2.3-", ".."} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("MustParseVersion should panic")
			}
		}()
		MustParseVersion("nope")
	}()
}

func TestCompareAndOps(t *testing.T) {
	a, b := MustParseVersion("1.2.3"), MustParseVersion("1.2.4")
	if !a.LT(b) || !b.GT(a) || !a.LTE(a) || !a.GTE(a) || !a.EQ(a) || !a.NEQ(b) {
		t.Error("basic ops")
	}
	if a.Compare(b) != -1 || b.Compare(a) != 1 || a.Compare(a) != 0 {
		t.Error("compare returns")
	}
	// differing lengths: 1.2 vs 1.2.0 are equal
	if MustParseVersion("1.2").Compare(MustParseVersion("1.2.0")) != 0 {
		t.Error("length-normalised compare")
	}
	// CalVer (major > 1996) sorts BEFORE SemVer
	cal, sem := MustParseVersion("20230101"), MustParseVersion("1.0.0")
	if !cal.LT(sem) {
		t.Error("calver should sort before semver")
	}
}

func TestString4CompAndPretty(t *testing.T) {
	if MustParseVersion("1.2.3.4").String() != "1.2.3.4" {
		t.Error("4-component join")
	}
	if MustParseVersion("2").String() != "2.0.0" {
		t.Error("short canonical")
	}
}
