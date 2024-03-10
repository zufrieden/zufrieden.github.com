---
title: "Fresh install - OSX "
date: 2021-01-01T14:13:38+02:00
showDate: true
draft: false
tags: ["geek"]
description : As a reminder to my future self, here is my personal recipe for new fresh install
---

As a reminder to my future self, here is my personal recipe for new fresh install. Yes it looks like holiday is a great time to update OSX to the new and shiny one.

## Install OSX - then re-install OSX
Format the hard drive with Disk Utility between the 2 installs
`⌘ + R` on startup

## Get .dotfile and .ssh keys
From only you know where :-) and set right permissions
* .ssh directory: 700 (drwx------)
* public key (.pub file): 644 (-rw-r--r--)
* private key (id_rsa): 600 (-rw-------)
* lastly your home directory should not be writeable by the group or others (at most 755 (drwxr-xr-x)).

## Sync iCloud 
- Careful with the 2 accounts (1 coming from a previous era of .Mac account)

## Pre-essential 
- Download and install [iA Writer Mono](https://github.com/iaolo/iA-Fonts/tree/master/iA%20Writer%20Mono)
- Configure Mail.app with the font :-) and disable “load remote content”

### Remap capslock and alt+tab

1. Open System Preferences → Keyboard.
2. Shortcuts, Keyboard, Move focus to next window set alt+tab
2. On Keyboard, Click the Modifier Keys button in the bottom right-hand corner.
3. Click the drop down box next to the hardware key that you'd like to remap, and select Escape.
4. Click OK and close System Preferences.

## In Mac App Store
- Download previously bought apps (that I still need)

## Install MacPort
- [go to macports.org](https://www.macports.org/install.php)

## Install softs
```bash
sudo port install tig
sudo port install hugo
sudo port install woff2
```

## Workflows

* Alfred - [switch to darkmode](https://www.packal.org/workflow/dark-mode-toggle)


## Install fish
```bash
sudo port install fish
```

## Git config
```bash
touch ~/.gitignore
```

* add .DS_Store, .hugo_build.lock etc..

```bash
 git config --global core.excludesfile ~/.gitignore
```

[source](https://sebastiandedeyne.com/setting-up-a-global-gitignore-file/)

