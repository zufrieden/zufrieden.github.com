---
title: "Feature Detecting “Undetectable” CSS Features with @supports named-feature()"
date: 2026-08-28T21:06:29+02:00
showDate: false
draft: false
link: "https://www.bram.us/2026/08/27/feature-detecting-undetectable-css-features-with-supports-named-feature/"
tags: ["Chromium","Firefox","Safari","Google","WebKit","Blink","Gecko","web development","frontend","CSS","keep"]
description: "Traditionally, <a href=\"/tags/css\">CSS</a> developers use the `@supports` rule to detect browser support for properties, values, and selectors. However, checking for underlying implementation updates, behavior changes, or properties that suddenly work together has historically been impossible without hacky workarounds. To solve this problem, the CSS Working Group introduced `@supports named-feature()`, a specialized function designed to expose specific capabilities via predefined keywords. This feature addresses tricky edge cases where conventional feature detection fails, such as transform-aware anchor positioning and single-axis scroll containers. By utilizing specific keywords within the function, developers can accurately verify browser behaviors and ensure progressive enhancement across modern engines like <a href=\"/tags/chromium\">Chromium</a>, <a href=\"/tags/firefox\">Firefox</a>, and <a href=\"/tags/safari\">Safari</a>, ultimately improving the reliability of web layouts."
---
