# semver — pkgx-compatible versions & ranges for Go

A faithful Go port of [pkgx](https://pkgx.sh)'s version scheme (libpkgx's
`semver.ts`). It is deliberately **not** strict SemVer 2.0: the open-source world
ships many "almost valid" versions that node-semver and the Go semver libraries
(`Masterminds/semver`, `golang.org/x/mod/semver`, `blang/semver`) reject —

- `openssl 1.1.1w` — a trailing letter (→ its 1-based alphabet index)
- `ghc 5.64.3.2` — four components
- bare `v` prefixes

— plus pkgx's own `@` range operator and CalVer sorting. This is the shared
implementation for the [`go-pkgx`](https://github.com/go-pkgx) family
(bottle / pkgm / pkgx / mirror / bk), so every tool resolves versions identically.

## Usage

```go
v, _ := semver.ParseVersion("1.1.1w")   // openssl-style
r, _ := semver.ParseRange("^1.1")       // ^ ~ < = @ >=x<y * , ||
r.Satisfies(v)                          // true

a, _ := semver.NewRange("^1")
b, _ := semver.NewRange(">=1.2 <1.5")
c, _ := semver.Intersect(a, b)          // range algebra
```

- `ParseVersion` / `MustParseVersion` — the loose version parser.
- `Version.Compare` and `EQ/NEQ/GT/GTE/LT/LTE` — CalVer (major > 1996) sorts before SemVer.
- `NewRange` (strict) / `ParseRange` (tolerant: a bare `5.1` becomes `@5.1`).
- `Range.Satisfies`, `Range.Max`, `Range.Single`, `Range.String`, and `Intersect`.

100% statement coverage; builds `CGO_ENABLED=0` on every arch. BSD-3-Clause.
