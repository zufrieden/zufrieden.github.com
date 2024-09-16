---
title: "Useful GIT command I sometimes need"
date: 2024-09-15T16:18:35+02:00
showDate: true
draft: false
tags: ["blog","dev"]
description : Some git command I need the most
---

Instead of just searching the doc, here is what I need to remember :

## Rename a branch (and apply the new name remotely)

```
git branch -m OLD-BRANCH-NAME NEW-BRANCH-NAME
git fetch origin
git branch -u origin/NEW-BRANCH-NAME NEW-BRANCH-NAME
git remote set-head origin -a
git remote prune origin
```

[Docs on github](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-branches-in-your-repository/renaming-a-branch#about-renaming-branches)