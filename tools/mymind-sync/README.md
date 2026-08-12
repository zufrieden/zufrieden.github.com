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
| body     | the file, or else the link         | the file, or else empty         |
| files    | saved into the page                | saved into the page             |
| filename | `YYYYMMDD_slugified_title.md`      | `slugified_title.md`, no date   |

The `linking` filename carries no date prefix, matching the pages already in
`content/linking/`. The done tag is never included in the page's own `tags`.

## Files

An object saved as an upload — a PDF, an image — carries a `blob`, and often no
URL at all. Those used to be skipped for want of anything to link to. Now the
file *is* the page: both sections download it from `GET /objects/:id/blob` and
write a [leaf bundle](https://gohugo.io/content-management/page-bundles/)
instead of a plain page.

```
content/sharing/20260802_vers_une_societe_des_communs/
  index.md
  vers_une_societe_des_communs.pdf
```

The URL is unchanged — Hugo renders `thing.md` and `thing/index.md` at the same
address — but a file inside a bundle is a *page resource*, and that is the only
thing [Hugo's image processing](https://gohugo.io/content-management/image-processing/)
works on. Anything under `static/` it can only copy. So the theme's image render
hook (`layouts/_default/_markup/render-image.html`) resizes a bundled image to
1200px, serves it as WebP with a 2x `srcset`, and passes every other image
through untouched — including the `/images/mastodon/…` paths that
[`mastodon-sync`](../mastodon-sync/) writes into `static/`.

| the object                | the body                                         |
| ------------------------- | ------------------------------------------------ |
| an image                  | `![title](photo.jpg)`                             |
| any other file            | `[the uploaded name.pdf](the_uploaded_name.pdf)`  |
| a file *and* a source URL | the file alone (`sharing`)                        |

In a `sharing` body the file and the link never appear together. An object's
`source.url` is where mymind fetched the bytes from, so for an upload it is the
file's own address on somebody's CDN — a link that would send the reader to the
very thing the page is already showing them. For the same reason the file's
name, not that URL's host, is what names an untitled page.

On a `linking` page the two do not compete: the link lives in the `link:` front
matter key and the file in the body, so an object carrying both keeps both. An
object with only a file gets a bundle and an empty `link: ""` — written
explicitly, because the archetype ships a placeholder `link: https://` that
would otherwise survive and render a 🌍 link to nowhere.

The saved filename is derived, never taken: the uploaded name is slugified (the
object's title stands in when mymind kept no name), and the extension comes from
the MIME type. The type itself is read from the object, then from the download's
`Content-Type`, and failing both from the bytes — mymind's media host serves
uploads as `application/octet-stream`, which would otherwise leave an image
looking like an anonymous file to link rather than show.

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
- **Two items shared the same day** with the same slug get a `_2`, `_3`, …
  suffix. A page and a bundle of the same name compete for it: Hugo renders both
  at the same URL and refuses to build a site holding the two.
- **Objects with neither a link nor a file are skipped.** A plain note tagged
  `share` or `keep` has nothing to publish. It is reported and left **untagged**, so it
  gets picked up again once it has one. A skip is not a failure — the rest of
  the batch still publishes.
- **A file that will not download fails its object rather than publishing
  without it.** A page missing the very thing it is about is worse than no page;
  nothing is tagged, so the next run retries from scratch. Rolling back a bundle
  removes the directory, file included.

## New to Go?

[`GO-PRIMER.md`](GO-PRIMER.md) walks through the language concepts used here —
packages, imports, pointers, interfaces, error handling and the testing
conventions — with every example pointing at real code in this directory.

## Adding a content type

`internal/publisher/mapping.go` holds a registry of `Mapping` values. A mapping
declares the Hugo section, the trigger tag, the done tag, and a `Build` function
turning an object into a `Page`. `Page.Extra` carries section-specific front
matter — for `linking` that is the archetype's `tags:` and `link:` keys.
Register the new mapping and it becomes available as `-type <name>`.

## API notes

- Docs: <https://access.mymind.com/api>
- Auth: a fresh HS256 JWT per request, `kid` = key id, claims pin the token to
  the exact `path` and `method` (see `internal/mymind/client.go`).
- The key secret is **base64-encoded key material**: it is decoded to raw bytes
  before signing. Signing with the base64 text instead gets you a
  `401 Invalid signature`. Paste the secret as shown — the tool decodes it.
- The `path` claim is the path only; the query string is not signed.
- Endpoints used: `GET /objects` (with a `q` search query),
  `GET /objects/:id/blob` and `POST /objects/:id/tags`.
- The blob endpoint answers with the bytes or a redirect to mymind's media host.
  The token is minted for one path on the API host, so it is stripped before the
  redirect is followed — from there the pre-signed URL is the credential.
- The API is in beta; the client decodes object payloads leniently (JSON array,
  NDJSON stream, or enveloped array) so a response-shape change is less likely
  to break the run.
