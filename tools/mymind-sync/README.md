# mymind-sync

Publishes [mymind](https://mymind.com) objects to this Hugo site.

Once a day (or on demand) it looks for objects carrying a trigger tag, creates
the matching Hugo page, and then writes a "done" tag back to mymind so nothing
is ever published twice.

| mymind tag | Hugo section       | done tag |
| ---------- | ------------------ | -------- |
| `share`    | `content/sharing/` | `shared` |
| `keep`     | `content/linking/` | `keeped` |

## Mapping

Common to both:

| Hugo front matter | mymind object                       |
| ----------------- | ----------------------------------- |
| `title`           | `title`                             |
| `date`            | `created` (the day it was saved)    |
| `description`     | first `notes` entry, else `summary` |

The date is the object's mymind `created` timestamp, read in `-timezone`, not
the day the tool happens to run — so a late or catch-up run still dates each
page by when you saved it. Objects whose `created` is missing or unparseable
fall back to the run date.

Where they differ:

|          | `sharing`                          | `linking`                       |
| -------- | ---------------------------------- | ------------------------------- |
| the URL  | body, as a markdown link           | `link:` front matter key        |
| `tags`   | —                                  | the object's mymind tags        |
| body     | the markdown link                  | empty                           |
| filename | `YYYYMMDD_slugified_title.md`      | `slugified_title.md`, no date   |

The `linking` filename carries no date prefix, matching the pages already in
`content/linking/`. The done tag is never included in the page's own `tags`.

Pages are scaffolded with `hugo new content content/<section>/<file>.md`, so
the archetypes stay the source of truth for `showDate`, `draft` and anything
else you add there. Only the mapped keys are overwritten afterwards.

## Setup

```sh
cd tools/mymind-sync
cp .env.example .env
# paste a FULL ACCESS key from https://access.mymind.com/extensions
```

`.env` is gitignored. Real environment variables always take precedence over
`.env`, which is how the GitHub Action supplies repository secrets.

## Usage

```sh
cd tools/mymind-sync

go run . -dry-run -v         # show what would be published, touch nothing
go run . -v                  # publish both sections for real
go run . -v -type linking    # just one section
cd .. && go test ./...       # tests for every tool, including shared hugosite
```

`content/sharing/` is also fed by [`mastodon-sync`](../mastodon-sync/), which
publishes Mastodon posts into the same section. The two never collide: they key
off different tags and record different front matter.

The Hugo site root is discovered by walking up from `-repo` (default: the
current directory), so running from `tools/mymind-sync` just works.

### Flags

| Flag         | Default            | Meaning                                            |
| ------------ | ------------------ | -------------------------------------------------- |
| `-type`      | `linking,sharing`  | Comma-separated content types; both run by default |
| `-repo`      | `.`              | Path inside the Hugo site                            |
| `-env`       | `.env`           | Credentials file (optional)                          |
| `-query`     | *(mapping's)*    | Override the mymind search query                     |
| `-limit`     | `50`             | Max objects per run                                  |
| `-timezone`  | `Europe/Zurich`  | Timezone `created` is read in for the date and prefix |
| `-hugo`      | `hugo`           | Hugo executable used for scaffolding                 |
| `-dry-run`   | `false`          | Report only; no files written, no tags added         |
| `-v`         | `false`          | Verbose logging                                      |

## GitHub Actions

`.github/workflows/mymind-sync.yml` runs the same command on a schedule.

**One-time setup** — add two repository secrets under
*Settings → Secrets and variables → Actions*:

| Secret | Value |
| ------ | ----- |
| `MYMIND_KEY_ID` | The key id from https://access.mymind.com/extensions |
| `MYMIND_PRIVATE_KEY` | That key's secret |

**Schedule.** `0 5 * * *` — 05:00 UTC, so 07:00 in Zurich during summer time and
06:00 in winter. Change the `cron:` line to move it.

**What a run does:**

1. Checks out `source`, installs Go and Hugo.
2. Runs `mymind-sync`, which creates pages and tags the objects `shared`.
3. Commits anything new to `content/sharing/` as `github-actions[bot]` and
   pushes to `source`.
4. Calls `ftp.yml` to build and deploy.

Step 4 exists because a push made with `GITHUB_TOKEN` does **not** trigger
`ftp.yml`'s own push trigger, so the deploy has to be invoked explicitly.

If some objects fail while others succeed, the successful pages are still
committed and deployed; the job is marked failed so you get a notification.

**Testing it by hand.** Go to *Actions → mymind sync → Run workflow*. Tick
**dry run** to see what it would publish without writing files or touching your
mymind tags. Leave it unticked to publish for real.

## Behaviour worth knowing

- **Tagging is the bookkeeping, the page is the deliverable.** The page is
  written first, then the object is tagged. If tagging fails the page is rolled
  back, so the next run retries cleanly instead of publishing a second copy
  under a `_2` suffix.
- **The object's tags are the authority.** Even if the search query misbehaves,
  anything already carrying `shared` is skipped.
- **Two items shared the same day** with the same slug get a `_2`, `_3`, … suffix.
- **Objects with no link are skipped.** A sharing page *is* a link, so a PDF,
  image, file or plain note tagged `share` has nothing to point at. It is
  reported and left **untagged**, so it gets picked up again if you later add a
  URL. A skip is not a failure — the rest of the batch still publishes.

## New to Go?

[`GO-PRIMER.md`](GO-PRIMER.md) walks through the language concepts used here —
packages, imports, pointers, interfaces, error handling and the testing
conventions — with every example pointing at real code in this directory.

## Adding a content type

`internal/publisher/mapping.go` holds a registry of `Mapping` values. A mapping
declares the Hugo section, the trigger tag, the done tag, and a `Build` function
turning an object into a `Page`. `Page.Extra` carries section-specific front
matter — for a future `linking` type that would be the archetype's `link:` key.
Register the new mapping and it becomes available as `-type <name>`.

## API notes

- Docs: <https://access.mymind.com/api>
- Auth: a fresh HS256 JWT per request, `kid` = key id, claims pin the token to
  the exact `path` and `method` (see `internal/mymind/client.go`).
- The key secret is **base64-encoded key material**: it is decoded to raw bytes
  before signing. Signing with the base64 text instead gets you a
  `401 Invalid signature`. Paste the secret as shown — the tool decodes it.
- The `path` claim is the path only; the query string is not signed.
- Endpoints used: `GET /objects` (with a `q` search query) and
  `POST /objects/:id/tags`.
- The API is in beta; the client decodes object payloads leniently (JSON array,
  NDJSON stream, or enveloped array) so a response-shape change is less likely
  to break the run.
