// Package semver is a faithful Go port of pkgx's version scheme (libpkgx's
// semver.ts). It is deliberately NOT strict SemVer 2.0: open-source ships many
// "almost valid" versions that node-semver and the Go semver libraries reject —
// openssl 1.1.1w (a trailing letter), ghc 5.64.3.2 (four components), bare
// v-prefixes — plus pkgx's own `@` range operator and CalVer sorting. This is
// the shared implementation for the go-pkgx family (bottle/pkgm/pkgx/mirror/bk).
package semver

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// inf is the open upper bound used for `>=x` ranges (TS uses Infinity).
const inf = math.MaxInt

// Version is a parsed pkgx version. Components holds every numeric part (a
// trailing letter like the "w" in 1.1.1w becomes its 1-based alphabet index).
type Version struct {
	Components          []int
	Major, Minor, Patch int
	Raw                 string
	pretty              string
}

var (
	letteredRE = regexp.MustCompile(`^(\d+)([a-z])$`)
	numericRE  = regexp.MustCompile(`^\d+$`)
)

// ParseVersion parses a version string, tolerating pkgx's loose scheme.
func ParseVersion(input string) (*Version, error) {
	raw := strings.TrimPrefix(input, "v")
	parts := strings.Split(raw, ".")
	v := &Version{Raw: raw}
	prettyIsRaw := false
	for i, x := range parts {
		if m := letteredRE.FindStringSubmatch(x); m != nil {
			if i != len(parts)-1 {
				return nil, fmt.Errorf("invalid version: %s", input)
			}
			n, _ := strconv.Atoi(m[1])
			v.Components = append(v.Components, n, int(m[2][0])-'a'+1)
			prettyIsRaw = true
		} else if numericRE.MatchString(x) {
			n, _ := strconv.Atoi(x)
			v.Components = append(v.Components, n)
		} else {
			return nil, fmt.Errorf("invalid version: %s", input)
		}
	}
	if prettyIsRaw {
		v.pretty = raw
	}
	v.fill()
	return v, nil
}

// MustParseVersion is ParseVersion but panics on error — for constants/tests.
func MustParseVersion(input string) *Version {
	v, err := ParseVersion(input)
	if err != nil {
		panic(err)
	}
	return v
}

// fromComponents builds a Version directly from numeric components.
func fromComponents(cc []int) *Version {
	v := &Version{Components: append([]int(nil), cc...)}
	v.Raw = joinInts(cc)
	v.fill()
	return v
}

func (v *Version) fill() {
	v.Major = at(v.Components, 0)
	v.Minor = at(v.Components, 1)
	v.Patch = at(v.Components, 2)
}

// String renders the version: its pretty form (for lettered versions), else the
// canonical major.minor.patch, or the full component join for 4+ components.
func (v *Version) String() string {
	if v.pretty != "" {
		return v.pretty
	}
	if len(v.Components) <= 3 {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}
	return joinInts(v.Components)
}

// Compare returns -1, 0 or 1. Components beyond one operand default to 0, and a
// CalVer major (>1996) sorts before a SemVer major (pkgx labels pre-releases
// with CalVer).
func (v *Version) Compare(that *Version) int {
	a, b := cmpComponents(v), cmpComponents(that)
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		c, d := at(a, i), at(b, i)
		if c != d {
			if c < d {
				return -1
			}
			return 1
		}
	}
	return 0
}

func cmpComponents(v *Version) []int {
	if v.Major > 1996 && v.Major != inf {
		return append([]int{0, 0, 0}, v.Components...)
	}
	return v.Components
}

// Comparison helpers.
func (v *Version) EQ(o *Version) bool  { return v.Compare(o) == 0 }
func (v *Version) NEQ(o *Version) bool { return v.Compare(o) != 0 }
func (v *Version) GT(o *Version) bool  { return v.Compare(o) > 0 }
func (v *Version) GTE(o *Version) bool { return v.Compare(o) >= 0 }
func (v *Version) LT(o *Version) bool  { return v.Compare(o) < 0 }
func (v *Version) LTE(o *Version) bool { return v.Compare(o) <= 0 }

func at(xs []int, i int) int {
	if i < len(xs) {
		return xs[i]
	}
	return 0
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return strings.Join(parts, ".")
}
