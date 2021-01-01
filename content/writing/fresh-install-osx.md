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

## Install Homebrew
`/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`

## Add cask
`brew tap homebrew/cask`

## Install softs
`brew install --cask finicky`

`brew install --cask beaker-browser`

`brew install --cask homebrew/cask-versions/firefox-developer-edition`

`brew install --cask hazel`

`brew install --cask firefox`

`brew install --cask doxie`

`brew install --cask authy`

`brew install --cask nova`

`brew install --cask alfred`

`brew install --cask docker`

`brew install --cask transmit`

`brew install --cask vlc`

`brew install fish`

`brew install --cask sequel-pro`

`brew install tig`

`brew install --cask sitesucker`

`brew install --cask Transmission`

`brew install hugo`


## Install oh my zsh
`sh -c "$(curl -fsSL https://raw.github.com/robbyrussell/oh-my-zsh/master/tools/install.sh)"`


