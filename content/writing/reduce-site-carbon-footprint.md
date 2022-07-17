---
title: "Reduce the carbon footprint of this website 🌱"
date: 2022-04-12T22:28:03+02:00
showDate: true
draft: false
description: How this website was designed and coded with sustainability in mind
tags: ["blog","dev","sustainability"]
---

## Why should I reduce the energy this website consume

Well if you're not aware of the current world situation, you might start by subscribing to [Zopeful newsletter course](https://zopeful.com). Any action to reduce energy consumption is good. Even a small drop is already a small drop. I know this website doesn't have a wide audience (well, I don't know, there is no tracking script). But making this website more efficient helps me progress in my craft.

## How I designed and coded this website

Designing this website was all about my dear web concept URL are forever. And static site are generated to live longer. Using [Hugo](https://gohugo.io) with a custom made template, a minimalist style, allowing 2 fonts and no images. I decided to use emoji but no images. Keep it simple, accessible and text oriented is already a big win over bloated websites. I also decided not to use Javascript but the [Service Workers](../going-offline-is-great).

## Before and after optimising

It's really nice to go on this path, but we need some numbers before and after. The good news is that before optimisation, the homepage get a "Hurrah! This web page is cleaner than 97%  of web pages tested". But there is still work to do to optimise.

## Let's dig into the different optimisations made this year

This website source code is available on github and the [complete commit history](https://github.com/zufrieden/zufrieden.github.com/commits/source) too. There is no secret to hide, sharing is also part of [the web I love](../a-love-letter-to-the-web).

## Compressing images, and use only small sizes in the design. Hosting provider constraints

By highly [compress images](https://github.com/zufrieden/zufrieden.github.com/commit/ba8f2fdb99f8aea9d424849abcbefb4c070c4663) and resize photos, I am able to keep the whole website under 10mo. The website is hosted at [infomaniak](https://www.infomaniak.com/en/hosting/web-and-mail) which offer a free static site hosting under 10mo. Infomaniak also provide a sustainable hosting offer (with a real CO2 roadmap and compensation).

## Lazyload images and font loading and service workers
Using modern web technics to enhance the performance is also a wining point for a more optimized website in term of energy efficiency. The website is using [font londing technics](../lighthouse-speed-test-and-fonts/) and images lazy loading with ```<img loading="lazy"/>``` and local caching for assets with [Service Workers](../going-offline-is-great).

## Further reading

- https://gerrymcgovern.com/designing-sustainable-websites/
- https://www.wholegraindigital.com/blog/website-energy-efficiency/
- https://gomakethings.com/your-website-is-a-pollution-machine/