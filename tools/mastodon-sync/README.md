# mastodon-sync

Publishes [Mastodon](https://social.coop/@zufrieden) posts to this Hugo site as
`content/sharing/` pages.

Once a day (or on demand) it reads the account's public RSS feed, skips every
post it has already published, and creates a page for the rest — with a copy of
any attachments under `static/`.

No credentials: the feed is public.

## Mapping

A Mastodon post has no title — the text *is* the content — so the mapping is not
quite the one `mymind-sync` uses for the same section:

| Hugo front matter | Mastodon post                                          |
| ----------------- | ------------------------------------------------------ |
| `title`           | derived from the post (see below)                      |
| `date`            | `pubDate`, read in `-timezone`                         |
| `description`     | the post as plain text, on one line                    |
| `mastodon_id`     | the status id — **this is what prevents duplicates**   |
| `mastodon_url`    | the permalink back to the post                         |
| body              | the post as markdown, then its attachments as images   |

```
---
title: "Sunset in the Baltic Sea"
date: 2026-08-09T17:58:45+02:00
showDate: true
draft: false
description: "Sunset in the Baltic Sea"
mastodon_id: "117066375180847363"
mastodon_url: "https://social.coop/@zufrieden/117066375180847363"
---
Sunset in the Baltic Sea

![Sunset in the Baltic Sea](/images/mastodon/117066375180847363/1.jpeg)
```

Filenames follow the section's convention, `YYYYMMDD_slugified_title.md`. Two
posts from the same day with the same slug get a `_2`, `_3`, … suffix.

**The title** is the first line of the post that says something, with links
dropped (a post usually names its link inline — "Yes https://…" — and the URL is
already in the body) and quotation markers (`>`, `"`, `“`) trimmed. It is cut to
70 characters on a word boundary and marked with `…`. A post with nothing but
pictures becomes "Photo", "Video" or "Audio".

**The body** keeps the post's real links rather than Mastodon's shortened display
text, and its line breaks: Goldmark is not configured with `hardWraps`, so a
`<br>` becomes an explicit `\` hard break.

**Attachments** are downloaded into `static/images/mastodon/<status-id>/1.jpeg`,
`2.jpeg`, … and referenced as `/images/mastodon/<status-id>/1.jpeg`. The feed
carries no alt text, so the derived title stands in.

Pages are scaffolded with `hugo new content content/sharing/<file>.md`, so
`archetypes/sharing.md` stays the source of truth for `showDate`, `draft` and
anything else you add there. Only the mapped keys are overwritten afterwards.

## How duplicates are avoided

mymind-sync tags the object it has published, so the record lives at the source.
An RSS feed is read-only — there is nowhere to write that back — so the record
lives in the site instead:

1. Every generated page carries `mastodon_id` in its front matter.
2. Each run scans `content/sharing/**` front matter and collects those ids.
3. Any post whose id is already there is skipped.

The page and its own "already published" marker are therefore the same file,
committed together. Unlike a state file they cannot drift apart, and unlike a
"newest date seen" watermark a failed run leaves no gap that can never be
filled.

Two consequences worth knowing:

- **A hand-written page counts.** Add `mastodon_id: "1234"` to any page in
  `content/sharing/` and that post will never be published again. This is the
  escape hatch if a post's attachment is permanently gone and the run keeps
  failing on it.
- **Deleting a page un-publishes it.** The next run will create it again, as long
  as the post is still in the feed.

## Usage

```sh
cd tools/mastodon-sync

go run . -dry-run          # show what would be published, touch nothing
go run . -v                # publish for real
go run . -v -limit 5       # only the 5 newest posts
cd .. && go test ./...     # tests for every tool
```

The Hugo site root is discovered by walking up from `-repo` (default: the current
directory), so running from `tools/mastodon-sync` just works.

### Flags

| Flag        | Default                              | Meaning                                              |
| ----------- | ------------------------------------ | ---------------------------------------------------- |
| `-feed`     | `https://social.coop/@zufrieden.rss` | Account RSS feed to read                             |
| `-repo`     | `.`                                  | Path inside the Hugo site                            |
| `-limit`    | `0`                                  | Max posts to consider, newest first (0 = whole feed)  |
| `-timezone` | `Europe/Zurich`                      | Timezone `pubDate` is read in, for the date and prefix |
| `-hugo`     | `hugo`                               | Hugo executable used for scaffolding                 |
| `-dry-run`  | `false`                              | Report only; nothing written or downloaded            |
| `-v`        | `false`                              | Verbose logging                                       |

## GitHub Actions

`.github/workflows/mastodon-sync.yml` runs the same command on a schedule. There
are no secrets to configure.

**Schedule.** `30 5 * * *` — 05:30 UTC, half an hour after `mymind-sync`, so the
two never race for the same `content/sharing` commit.

**What a run does:**

1. Checks out `source`, installs Go and Hugo.
2. Runs `mastodon-sync`, which creates pages and downloads attachments.
3. Commits anything new in `content/sharing/` and `static/images/mastodon/` as
   `github-actions[bot]` and pushes to `source`.
4. Calls `ftp.yml` to build and deploy.

Step 4 exists because a push made with `GITHUB_TOKEN` does **not** trigger
`ftp.yml`'s own push trigger, so the deploy has to be invoked explicitly.

**Testing it by hand.** *Actions → mastodon sync → Run workflow*. Tick **dry
run** to see what it would publish without writing anything.

## Behaviour worth knowing

- **The feed only carries the 20 most recent posts.** Anything older cannot be
  backfilled this way. The public REST API
  (`/api/v1/accounts/:id/statuses`, no auth needed) pages further back if you
  ever want the whole history.
- **Posts are published whole or not at all.** If an attachment cannot be
  fetched, the page is not written and the downloaded files are removed. Since
  the id is only recorded on a page that exists, the next run retries cleanly.
  The post is reported as *failed*, which marks the job red so you notice.
- **Edits are not synced.** Editing a post on Mastodon keeps the same status id,
  so an existing page is left alone. Delete the page if you want it regenerated.
- **Boosts and replies never appear**, because Mastodon leaves them out of the
  account feed.
- **Posts are published oldest-first** within a run, so same-day `_2` suffixes
  follow the order the posts were actually made.

## Layout

```
tools/
  go.mod                    one module for every tool
  internal/hugosite/        shared: site discovery, front matter, slugs
  mastodon-sync/
    main.go
    internal/mastodon/      RSS client, and Mastodon's HTML -> text/markdown
    internal/publisher/     post -> page mapping, dedupe, attachments
```

`internal/hugosite` is shared with `mymind-sync`; everything under
`mastodon-sync/internal/` is private to this tool. See
[`../mymind-sync/GO-PRIMER.md`](../mymind-sync/GO-PRIMER.md) if the Go is
unfamiliar.
