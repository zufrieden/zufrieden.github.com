---
title: "Optimizing font loading"
date: 2021-01-04T22:55:39+01:00
showDate: true
draft: false
tags: ["blog","dev"]
description : How to increase performance regarding font loading
---

## Lot of solutions to choose from

I came accros this great article on css-tricks on [how to load font in a way that fight FOUT and makes lighthouse happy](https://css-tricks.com/how-to-load-fonts-in-a-way-that-fights-fout-and-makes-lighthouse-happy/). I'm not here to please lighthouse, but what could be improved to speed up this website is welcomed. The solution explained here is to use some JS to wait rendering in order to load the font. But then a fallback is needed to have a non-obstructive solution. 

## Best build tool is no build tool

Love the idea that, for this sort of website, having a bare simple almost raw website is the best resilient approach. As posted here by Steren [My stack will outlive yours](https://blog.steren.fr/2020/my-stack-will-outlive-yours/). I loved the idea of choosing the lightest solution.

## Choose a fallback font

This tool [Font-style-matcher](https://meowni.ca/font-style-matcher/) helped me find the best system font alternative to avoid my page jumping to much when the font would load. 

## Font-display `swap`, `block`, `fallback`, `optional`

Another detailed article [on font loading from Nick Bull](https://nickbulljs.medium.com/3-css-font-properties-you-need-to-use-in-2020-1e60c415b8a8) explain different strategy on `font-display` usage.
I decide to go `fallback`. 

> `font-display: fallback;`  Acts as a compromise between the auto and swap values. The browser hides the text for about 100ms, and if the font has not yet been downloaded, the browser uses the fallback text. It will swap to the new font after it is downloaded, but only during a short swap period (probably three seconds).

As the font is loaded and cached by [the service worker](going-offline-is-great.md), on the next page, the user will have the font loaded anyway.



