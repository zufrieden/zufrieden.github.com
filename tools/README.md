# tools

Small Go programs that publish content onto this Hugo site from somewhere else.
Each runs locally and on a daily GitHub Actions schedule.

| Tool                                 | Source                       | Lands in                                | Credentials      |
| ------------------------------------ | ---------------------------- | --------------------------------------- | ---------------- |
| [`mymind-sync`](mymind-sync/)         | mymind objects tagged in-app | `content/sharing/`, `content/linking/`  | a mymind API key |
| [`mastodon-sync`](mastodon-sync/)     | the account's RSS feed       | `content/sharing/`                      | none (public)    |

## One module, shared parts

```
tools/
  go.mod                    module .../tools — one module, all tools
  internal/hugosite/        shared: site discovery, front matter, slugs, page writing
  mymind-sync/
    main.go
    internal/{config,mymind,publisher}/
  mastodon-sync/
    main.go
    internal/{mastodon,publisher}/
```

`internal/hugosite` is the shared part: everything about *writing a page into a
Hugo site* — finding the site root, scaffolding from the archetype, patching
front matter, slugifying a title, picking a free filename. It is the code where a
bug would quietly corrupt front matter, so there is one copy.

Each tool's own `internal/` stays private to it, and Go enforces that: nothing
outside `tools/mymind-sync/` can import `tools/mymind-sync/internal/...`. The
publishing pipelines are deliberately *not* shared — mymind-sync marks an object
as done by writing a tag back to mymind, mastodon-sync cannot write anything back
at all and records the status id in the page instead. A common abstraction would
be an interface with half its methods stubbed.

## Everyday commands

```sh
cd tools
go build ./...        # compile every tool
go test ./...         # test every tool
go vet ./...
gofmt -l .            # should print nothing

cd mastodon-sync && go run . -dry-run -v -repo ..
cd mymind-sync   && go run . -dry-run -v -repo ..
```

New to Go? [`mymind-sync/GO-PRIMER.md`](mymind-sync/GO-PRIMER.md) walks through
every language concept these tools use, pointing at real code.
