---
title: "Adding dark mode to this website"
date: 2021-01-02T22:06:08+01:00
showDate: true
draft: false
tags: ["blog","dev"]
description : "Because why not, that was actually quite easy"
---

A few line of CSS and a nice excuse to experiment with variables. [Full commit available on github](https://github.com/zufrieden/zufrieden.github.com/commit/30f011b5681705aacbe1b95bae6edd12c5382e52). I didn't want to add any JS to switch modes. I really like the fact that the toggle is available on the OS level. But Chris wrote a [nice tutorial on creating an automatic night mode with vanilla JS](https://gomakethings.com/automatic-night-mode-with-vanilla-js/).

## A few line of CSS

Simply adding a variable to define background color

```CSS
:root {
  --bg-color: hsla(21, 52%, 94%, 1);
}
```

And override on a media query for user requesting a darker color-scheme

```CSS
@media (prefers-color-scheme: dark) {
  :root {
	--bg-color: hsla(202, 85%, 4%, 1);
  }
}
```

## The real difficulty is to choose the color wisely
Using HSLA way of defining colors helps choose the right brightness as dark theme tend to look better with more vivid colors. This article by Tunghsiao Liu about [CSS Variables Guide: Color Manipulation and Dark Mode](https://sparanoid.com/note/css-variables-guide/) explain why and some tricks to ease the seek of the good color.

More ressources from [Robin Rendle on CSS Tricks](https://css-tricks.com/dark-modes-with-css/).