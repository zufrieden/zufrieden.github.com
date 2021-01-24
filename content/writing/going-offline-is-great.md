---
title: "Going offline is so great"
date: 2021-01-03T14:19:29+01:00
showDate: true
draft: false
tags: ["blog","dev"]
description : "How I added a service worker on this website"
---

## You might be offline, check your ✈️ mode.

You would see this page if my website is offline, or you are disconnected from the network, or you just click to see this article.

## Service Workers should be implemented based on the website strategy

I heard about Service Workers and PWA a few years ago. I really am appealed by this kind of technology. Simple, resilient, not relying on framework or 4 abstraction levels. After reading [Going Offline book](https://abookapart.com/products/going-offline) from Jeremy Keith more than a year ago I wanted to implement an offline strategy on my website. You could argue that your framework, CMS or generator implement service worker for you, so you don't have to worry. Wrong! You should consider your users and their cache capacity by adapting your service worker strategy according to it.

## Here is my strategy for zufrieden.io

I value your local memory, so I decided to cache 1 file of CSS, 2 fonts and this about offline mode article. The rest of the caching strategy is around your visit. Each page you load (up to 40 pages, 50 images), will be cached for 5 minutes. When the server is not in reach, you will have the page you requested. If not available in your cache, I will give you this article.

## how it's made

The full file is available here [/zufrieden-sw.js](/zufrieden-sw.js) for you to see. This code is a shameless copy from [adacio.com](https://adactio.com/serviceworker.js). Code explained on the book and also available with steps on [github/adacio/goingofflinecode](https://github.com/adactio/goingofflinecode).

In the `<head>` First register the service worker 

```JS
// Register our service-worker
if (navigator.serviceWorker) {
	window.addEventListener('load', function() {
		navigator.serviceWorker.register('/zufrieden-sw.js', {
			scope: '/'
		});
	});
}
```


## Listening native events `install`, `activate`, `fetch`.

Service worker is not altering the page first display performance as there is no relation with the DOM or the current page rendering. It will be installed, perform install event (caching CSS, fonts and this page then the current page). 

```js
addEventListener('fetch', function (event) {
  console.log('The service worker is listening.');
});

addEventListener('install', function (event) {
  console.log('The service worker is installing . . . ');
});

addEventListener('activate', function (event) {
  console.log('The service worker is activated.');
});
```

## Offline only website, why not

More offline project to discover : [The Disconnect](https://thedisconnect.co) is a great exemple of playing with service worker strategy.

## Still offline, check my love letter to the web
Building a more resilient web, understanding the value of the web and how we build a more sustainable heritage. I consider myself a web dinosaur, let's not condemn ourself to quickly to extinction. [Read my article about why I love the web](/writing/a-love-letter-to-the-web/)

