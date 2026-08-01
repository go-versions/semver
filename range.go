package semver

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// bound is one term of a Range: either an inclusive-exclusive interval [Lo, Hi)
// (Pair) or an exact single version (Single).
type bound struct {
	Lo, Hi *Version // interval when Single == nil
	Single *Version // exact-version term
}

// Range is a pkgx version constraint: `*`, `^1`, `~1.2`, `<3`, `=1.2.3`, `@5`,
// `>=1 <2`, or several of those joined by `,` / `||`.
type Range struct {
	star bool
	set  []bound
}

var (
	splitRE   = regexp.MustCompile(`(?:,|\s*\|\|\s*)`)
	gerange   = regexp.MustCompile(`^>=((?:\d+\.)*\d+)\s*(<((?:\d+\.)*\d+))?$`)
	oprange   = regexp.MustCompile(`^([~=<^@])(.+)$`)
	bareVerRE = regexp.MustCompile(`^(\d+\.)*\d+$`)
)

// NewRange parses a range string. It is strict: use Parse for the tolerant form.
func NewRange(input string) (*Range, error) {
	input = strings.TrimSpace(input)
	if input == "*" {
		return &Range{star: true}, nil
	}
	r := &Range{}
	for _, part := range splitRE.Split(input, -1) {
		b, err := parseBound(part)
		if err != nil {
			return nil, err
		}
		r.set = append(r.set, b)
	}
	for _, b := range r.set {
		if b.Single == nil && !b.Lo.LT(b.Hi) {
			return nil, fmt.Errorf("invalid semver range: %s", input)
		}
	}
	return r, nil
}

// ParseRange is the tolerant range parser: a bare version (eg. "5.1") that
// isn't a valid range is treated as `@5.1`, matching what users expect.
func ParseRange(input string) (*Range, error) {
	if input == "" {
		return nil, fmt.Errorf("empty range")
	}
	if r, err := NewRange(input); err == nil {
		return r, nil
	}
	if !bareVerRE.MatchString(input) {
		return nil, fmt.Errorf("invalid semver range: %s", input)
	}
	return NewRange("@" + input)
}

func parseBound(input string) (bound, error) {
	fail := func() (bound, error) { return bound{}, fmt.Errorf("invalid semver range: %s", input) }

	if m := gerange.FindStringSubmatch(input); m != nil {
		// the regex guarantees numeric operands, so parsing never fails
		lo, _ := ParseVersion(m[1])
		hi := fromComponents([]int{inf, inf, inf})
		if m[3] != "" {
			hi, _ = ParseVersion(m[3])
		}
		return bound{Lo: lo, Hi: hi}, nil
	}

	m := oprange.FindStringSubmatch(input)
	if m == nil {
		return fail()
	}
	rest := m[2]
	switch m[1] {
	case "^":
		v1, err := ParseVersion(rest)
		if err != nil {
			return fail()
		}
		var parts []int
		for i, c := range v1.Components {
			if c == 0 && i < len(v1.Components)-1 {
				parts = append(parts, 0)
			} else {
				parts = append(parts, c+1)
				break
			}
		}
		return bound{Lo: v1, Hi: fromComponents(parts)}, nil
	case "~":
		v1, err := ParseVersion(rest)
		if err != nil {
			return fail()
		}
		var hi *Version
		if len(v1.Components) == 1 {
			hi = fromComponents([]int{v1.Major + 1})
		} else {
			hi = fromComponents([]int{v1.Major, v1.Minor + 1})
		}
		return bound{Lo: v1, Hi: hi}, nil
	case "<":
		v2, err := ParseVersion(rest)
		if err != nil {
			return fail()
		}
		return bound{Lo: fromComponents([]int{0}), Hi: v2}, nil
	case "=":
		v, err := ParseVersion(rest)
		if err != nil {
			return fail()
		}
		return bound{Single: v}, nil
	default: // "@" — the only remaining operator the regex allows
		nums := strings.Split(rest, ".")
		parts := make([]int, len(nums))
		for i, s := range nums {
			n, err := strconv.Atoi(s)
			if err != nil {
				return fail()
			}
			parts[i] = n
		}
		v1 := fromComponents(parts)
		hi := append([]int(nil), parts...)
		hi[len(hi)-1]++
		return bound{Lo: v1, Hi: fromComponents(hi)}, nil
	}
}

// Satisfies reports whether version is within the range.
func (r *Range) Satisfies(version *Version) bool {
	if r.star {
		return true
	}
	for _, b := range r.set {
		if b.Single != nil {
			if version.EQ(b.Single) {
				return true
			}
			continue
		}
		if version.Compare(b.Lo) >= 0 && version.Compare(b.Hi) < 0 {
			return true
		}
	}
	return false
}

// Max returns the greatest version satisfying the range, or nil.
func (r *Range) Max(versions []*Version) *Version {
	var ok []*Version
	for _, v := range versions {
		if r.Satisfies(v) {
			ok = append(ok, v)
		}
	}
	if len(ok) == 0 {
		return nil
	}
	sort.Slice(ok, func(i, j int) bool { return ok[i].Compare(ok[j]) < 0 })
	return ok[len(ok)-1]
}

// Single returns the sole exact version a range denotes, or nil.
func (r *Range) Single() *Version {
	if r.star || len(r.set) != 1 {
		return nil
	}
	return r.set[0].Single
}

// String renders the range back to pkgx range syntax.
func (r *Range) String() string {
	if r.star {
		return "*"
	}
	parts := make([]string, len(r.set))
	for i, b := range r.set {
		parts[i] = b.String()
	}
	return strings.Join(parts, ",")
}

func (b bound) String() string {
	if b.Single != nil {
		return "=" + b.Single.String()
	}
	v1, v2 := b.Lo, b.Hi
	switch {
	case v2.Major == v1.Major+1 && v2.Minor == 0 && v2.Patch == 0:
		v := chomp(v1)
		if v1.Major == 0 {
			if len(v1.Components) == 1 {
				return "^0"
			}
			return ">=" + v + "<1"
		}
		return "^" + v
	case v2.Major == v1.Major && v2.Minor == v1.Minor+1 && v2.Patch == 0:
		return "~" + chomp(v1)
	case v2.Major == inf:
		return ">=" + chomp(v1)
	case atBound(v1, v2):
		return "@" + v1.String()
	default:
		return ">=" + chomp(v1) + "<" + chomp(v2)
	}
}

// atBound reports whether [v1,v2) is an `@`-style range (v2 == v1 with its last
// component incremented).
func atBound(v1, v2 *Version) bool {
	cc1 := append([]int(nil), v1.Components...)
	cc2 := v2.Components
	if len(cc1) > len(cc2) {
		return false
	}
	for len(cc1) < len(cc2) {
		cc1 = append(cc1, 0)
	}
	if cc1[len(cc1)-1] != cc2[len(cc2)-1]-1 {
		return false
	}
	for i := 0; i < len(cc1)-1; i++ {
		if cc1[i] != cc2[i] {
			return false
		}
	}
	return true
}

// Intersect returns the range satisfied by both a and b, or an error when they
// are disjoint.
func Intersect(a, b *Range) (*Range, error) {
	if b.star {
		return a, nil
	}
	if a.star {
		return b, nil
	}
	var set []bound
	for _, aa := range a.set {
		for _, bb := range b.set {
			switch {
			case aa.Single != nil && bb.Single != nil:
				if aa.Single.EQ(bb.Single) {
					set = append(set, aa)
				}
			case aa.Single != nil:
				if aa.Single.Compare(bb.Lo) >= 0 && aa.Single.LT(bb.Hi) {
					set = append(set, aa)
				}
			case bb.Single != nil:
				if bb.Single.Compare(aa.Lo) >= 0 && bb.Single.LT(aa.Hi) {
					set = append(set, bb)
				}
			default:
				if aa.Lo.Compare(bb.Hi) >= 0 || bb.Lo.Compare(aa.Hi) >= 0 {
					continue
				}
				lo, hi := bb.Lo, bb.Hi
				if aa.Lo.Compare(bb.Lo) > 0 {
					lo = aa.Lo
				}
				if aa.Hi.Compare(bb.Hi) < 0 {
					hi = aa.Hi
				}
				set = append(set, bound{Lo: lo, Hi: hi})
			}
		}
	}
	if len(set) == 0 {
		return nil, fmt.Errorf("cannot intersect: %s && %s", a, b)
	}
	return &Range{set: set}, nil
}

func chomp(v *Version) string {
	s := v.String()
	for strings.HasSuffix(s, ".0") {
		s = strings.TrimSuffix(s, ".0")
	}
	return s
}
