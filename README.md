# zufrieden.io

The source of [zufrieden.io](https://zufrieden.io/) — a [Hugo](https://gohugo.io/)
site, deployed over FTP by GitHub Actions.

The site is built from the `source` branch. Pushing to `source` builds and
deploys automatically; there is nothing to run by hand.

## Requirements

| Tool | Version | Needed for |
| ---- | ------- | ---------- |
| Hugo | `0.120.4` in CI, newer works locally | Building the site |
| Go   | `1.22+` | The `mymind-sync` tool only |

```sh
brew install hugo go
```

## Local development

```sh
hugo server -D          # http://localhost:1313, drafts included
hugo --minify           # production build into public/
```

## Writing content

Each section has an archetype in `archetypes/`, so use `hugo new` rather than
creating files by hand:

```sh
hugo new content content/sharing/20260810_some_title.md
hugo new content content/writing/some-post.md
```

| Section | Path | Archetype | What it holds |
| ------- | ---- | --------- | ------------- |
| Sharing | `content/sharing/` | `sharing.md` | Links worth passing on. Filenames are `YYYYMMDD_slugified_title.md`. |
| Writing | `content/writing/` | `writing.md` | Longer posts |
| Linking | `content/linking/` | `linking.md` | Link posts with a `link:` front matter key |
| Talking | `content/talking/` | `talking.md` | Talks |
| Making pictures | `content/making-pictures/` | — | Photo galleries |

Standalone pages (`content/about.md`, `content/now.md`) sit at the top level.

## Deployment

`.github/workflows/ftp.yml` runs on every push to `source`: it builds with
`hugo --minify` and uploads `public/` to `./sites/zufrieden.io/` over FTP.

Repository secrets required: `FTP_HOST`, `FTP_USERNAME`, `FTP_PASSWORD`.

The upload skips `sharing/200*`, `sharing/201*`, `sharing/2020*` and
`sharing/2021*` — the archive is already on the server and re-uploading two
thousand-odd pages on every deploy is slow.

The workflow is also callable from other workflows (`workflow_call`) with an
optional `ref` input, because a push made with `GITHUB_TOKEN` does not trigger
the push event above.

## Automation: publishing from mymind

`tools/mymind-sync/` is a small Go app that publishes
[mymind](https://mymind.com) objects to the site. Once a day it looks
for objects tagged `share` (→ `content/sharing/`) or `keep` (→ `content/linking/`)
and tags them `shared` / `keeped` in mymind so nothing is published twice.

Quick start:

```sh
cd tools/mymind-sync
cp .env.example .env      # paste a FULL ACCESS key from access.mymind.com/extensions
go run . -dry-run -v      # show what would be published, touch nothing
go run . -v               # publish for real
```

Full documentation — flags, the field mapping, the GitHub Actions setup, and how
to add a new content type — is in [`tools/mymind-sync/README.md`](tools/mymind-sync/README.md).

New to Go? [`tools/mymind-sync/GO-PRIMER.md`](tools/mymind-sync/GO-PRIMER.md)
explains the language concepts used in that app, with every example pointing at
real code in the project.

## Repository layout

```
archetypes/            front matter templates, one per section
content/               the site's content
themes/zufrieden002/   the active theme (see `theme` in config.toml)
static/                images, fonts, files served as-is
config.toml            site configuration
tools/mymind-sync/     mymind -> content/sharing publisher
.github/workflows/     ftp.yml (deploy), mymind-sync.yml (daily publish)
```
