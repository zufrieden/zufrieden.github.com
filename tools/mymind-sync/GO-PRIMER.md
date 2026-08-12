# Go, explained through this codebase

A walkthrough of the Go concepts used in `mymind-sync`, written for someone new
to the language. Every example points at real code in this directory rather
than a toy snippet, so you can open the file and read around it.

- [1. Module, packages, directories](#1-module-packages-directories)
- [2. Imports](#2-imports)
- [3. Capital letter = public](#3-capital-letter--public)
- [4. Syntax you'll trip on](#4-syntax-youll-trip-on)
- [5. Interfaces — the one genuinely different idea](#5-interfaces--the-one-genuinely-different-idea)
- [6. Tests](#6-tests)
- [Where to start reading](#where-to-start-reading)

---

## 1. Module, packages, directories

`tools/go.mod` declares the module. Its name is the import prefix for everything
inside:

```
module github.com/zufrieden/zufrieden.github.com/tools
go 1.22
```

One module covers every tool in `tools/`, which is what lets them share
`tools/internal/hugosite` without publishing it anywhere:

```
tools/
  go.mod
  internal/hugosite/        shared by every tool
  mymind-sync/
    main.go                 package main
    internal/{config,mymind,publisher}/
  mastodon-sync/
    main.go                 package main
    internal/{mastodon,publisher}/
```

Three rules that surprise most newcomers:

- **A directory is a package.** Every `.go` file in `internal/mymind/` starts
  with `package mymind` (`internal/mymind/types.go:1`,
  `internal/mymind/client.go:5`). They are one package split across files —
  `client.go` uses `Object` from `types.go` with no import.
- **`internal/` is enforced by the compiler**, relative to the directory it sits
  in. `tools/internal/hugosite` is importable by anything under `tools/`, while
  `tools/mymind-sync/internal/mymind` is importable only from inside
  `tools/mymind-sync/` — so `mastodon-sync` cannot reach into it even by
  accident. Nothing outside `tools/` can import either.
- **`package main` + `func main()` = an executable** (`main.go:12`,
  `main.go:32`). Every other package is a library.

## 2. Imports

```go
import (
	"context"       // stdlib: no prefix
	"os"
	_ "time/tzdata" // blank import

	"github.com/zufrieden/.../internal/publisher" // full path from go.mod
)
```

- You reference the **last path segment**: `publisher.Run(...)`, not the whole
  URL.
- **An unused import is a compile error**, not a warning. Go is aggressive
  about this; `gofmt` and your editor will strip them.
- `_ "time/tzdata"` (`main.go:24`) — the underscore means *"import for side
  effects only."* That package registers the world timezone database into the
  binary. Nothing in it is ever called, so without `_` it would not compile.
  It is what makes `-timezone Europe/Zurich` work on a bare CI runner.
- Standard library first, then a blank line, then external. `gofmt` maintains
  that grouping.

## 3. Capital letter = public

There is no `public` / `private` keyword. **The first letter decides:**

```go
func Run(...)          // publisher.go:59  — exported, callable from main.go
func publishOne(...)   // publisher.go:90  — package-private
type Object struct{}   // exported
baseURL string         // client.go:36 — lowercase field, invisible outside the package
```

This is why `Client` has lowercase fields (`client.go:35-41`) and a `New`
constructor (`client.go:48`): callers cannot reach in and set `secret`
directly, so `New` can guarantee it is always properly decoded.

## 4. Syntax you'll trip on

**Types come after names**, and reading right-to-left helps:

```go
var results []Result                              // slice of Result
func Count(results []Result, status Status) int   // params typed, then return type
tagged map[string][]string                        // map from string to slice-of-string
```

**`:=` declares and infers; `=` assigns to something that exists:**

```go
slug := collapseUnderscores(...)  // new variable, type inferred
slug = strings.Trim(slug, "_")    // reassign
```

**Multiple return values** — this is everywhere:

```go
objects, err := client.ListObjects(ctx, query, opts.Limit)
if err != nil {
	return nil, err
}
```

**There are no exceptions.** `error` is an ordinary return value, and
`if err != nil` is the whole error-handling story. That `if` block is why Go
code looks so vertical. `nil` is the zero value for errors, pointers, slices
and maps.

**Zero values, not null.** An unset `string` is `""`, an `int` is `0`, a struct
is a struct with all fields zeroed. `Result{}` is valid and usable immediately.

**`*` is a pointer.** `Source *Source` (`types.go:18`) is a pointer so it can be
`nil` — that is precisely how an uploaded PDF, which was never saved from a web
page and so has no source at all, is told apart (`types.go:162`):

```go
if o.Source != nil && strings.TrimSpace(o.Source.URL) != "" {
```

Order matters: `&&` short-circuits, so `o.Source.URL` is only evaluated once
`o.Source` is known non-nil. Reverse them and you get a nil-pointer panic on
the PDF. `Blob *Blob` is the same idea from the other side: nil means the object
carries no file, and `HasBlob` is that check under a name.

**Methods are functions with a receiver:**

```go
func (o Object) SourceURL() string   // types.go:161  — value receiver, gets a copy
func (c *Client) sign(...)           // client.go:96 — pointer receiver, can mutate
```

Rule of thumb: pointer receiver if the method mutates or the struct is large;
value receiver for small read-only ones. `UnmarshalJSON` *must* be a pointer
receiver (`types.go:111`) — it fills the struct in.

**Struct tags** are the string literals after fields (`types.go:15`):

```go
Title string `json:"title"`
```

They are metadata read at runtime by `encoding/json`. That is the entire
mapping between mymind's JSON and the Go struct — no manual parsing.

**`defer` runs when the function returns**, no matter which path (`client.go`,
in `do`):

```go
resp, err := c.http.Do(req)
if err != nil { return nil, err }
defer resp.Body.Close()   // guaranteed cleanup
```

**`for ... range` is the only loop.** No `while`, no `for(;;)`:

```go
for _, obj := range objects { ... }   // _ discards the index
for name := range registry { ... }    // over a map: keys only
```

`_` means "I must accept this value but do not want it." Unused *variables* are
also compile errors, so `_` gets used a lot.

## 5. Interfaces — the one genuinely different idea

`publisher.go:16`:

```go
type Fetcher interface {
	ListObjects(ctx context.Context, query string, limit int) ([]mymind.Object, error)
	Blob(ctx context.Context, objectID string) ([]byte, string, error)
	AddTags(ctx context.Context, objectID string, names ...string) error
}
```

Go interfaces are **implicit**. `*mymind.Client` never says "I implement
Fetcher" — it just happens to have all three methods, so it satisfies it. No
`implements` keyword, no inheritance.

Why this matters practically: `Run` takes a `Fetcher`, not a `*mymind.Client`.
So the tests hand it a `fakeClient` (`publisher_test.go:16-46`) with the same
three methods, and the whole publishing pipeline is testable **without a
network, an API key, or the real mymind account** — downloads included. That
single interface is why there are over fifty tests instead of three.

The `names ...string` is a *variadic* parameter — call it as
`AddTags(ctx, id, "shared")` or `AddTags(ctx, id, "a", "b")`; inside it is a
`[]string`.

## 6. Tests

Go has testing built into the toolchain — no framework, no assertion library.

**Conventions the tool enforces:**

- File must end `_test.go` (excluded from the real build).
- Function must be `func TestXxx(t *testing.T)`, capital after `Test`.
- Same package (`package publisher`) means tests can reach unexported things —
  that is how `hugosite_test.go` tests `patchFrontMatter` and
  `renderArchetype`, which are lowercase.

**No assertions — just `if` and a failure call:**

```go
if got, want := mapping.SearchQuery(), "tag:share -tag:shared"; got != want {
	t.Errorf("SearchQuery = %q, want %q", got, want)
}
```

- `t.Errorf` = record failure, **keep going**.
- `t.Fatal` / `t.Fatalf` = record failure, **stop this test now**. Use it when
  continuing would panic — after `err != nil`, or before indexing a slice whose
  length you just checked.
- `%q` quotes strings so you can see whitespace; `%v` is the general
  "print anything".
- The `got, want :=` inside the `if` is idiomatic — it scopes both to that
  statement.

**Table-driven tests** are the dominant Go style (`hugosite_test.go:11`):

```go
cases := map[string]string{
	"Good Design":  "good_design",
	"Café & Crème": "cafe_and_creme",
	"!!!":          "untitled",
}
for in, want := range cases {
	if got := Slugify(in); got != want {
		t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
	}
}
```

One loop, many cases; adding a case is one line. Note that map iteration order
is deliberately random in Go, so only use a map when the cases are independent.
`publisher_test.go:488` uses a slice of structs instead, where order and richer
fields matter.

**Helpers the standard library gives you:**

```go
root := t.TempDir()                 // temp dir, auto-deleted after the test
t.Setenv(EnvKeyID, "from-secrets")  // env var, auto-restored after the test
t.Helper()                          // in write(): report failures at the CALLER's line
```

**`httptest`** spins up a real HTTP server on a random port
(`internal/mymind/client_test.go`), so `TestListObjectsSignsAndQueries`
exercises genuine networking — building the request, signing the JWT, decoding
the response — against a stub that verifies the HMAC.

**Running them:**

```sh
go test ./...                     # every package
go test ./internal/publisher/     # one package
go test ./... -run TestSlugify -v # one test, verbose
go vet ./...                      # catches real bugs (bad Printf verbs, etc.)
gofmt -l .                        # lists misformatted files; -w rewrites them
```

There is one canonical formatting and no debate about it — always run `gofmt`.

## Where to start reading

`main.go:39` (`run`) is the entry point, and it is short: parse flags → load
config → build the client → call `publisher.Run` → print a report.

From there:

| Read next | Why |
| --------- | --- |
| `internal/publisher/publisher.go:59` | The loop over objects |
| `internal/publisher/publisher.go:90` | What happens to a single object, including the roll-back on a failed tag |
| `internal/publisher/mapping.go:79` | The mymind → Hugo field mapping, in one struct literal |
| `internal/mymind/client.go:96` | JWT signing — small, self-contained, and the part that had the base64 bug |
| `internal/mymind/types.go:105-159` | The trickiest code here: custom JSON unmarshalling that tolerates several response shapes |

## Further reading

- [A Tour of Go](https://go.dev/tour/) — interactive, an hour well spent
- [Effective Go](https://go.dev/doc/effective_go) — the idiom guide
- [Go by Example](https://gobyexample.com/) — short, searchable recipes
- [Table-driven tests](https://go.dev/wiki/TableDrivenTests)
