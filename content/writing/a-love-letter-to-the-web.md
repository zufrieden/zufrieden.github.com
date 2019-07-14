---
title: "Simple is the web, and why I love it so much."
date: 2019-07-08T16:23:49+02:00
showDate: true
draft: false
tags: ["blog","web"]
description : Through my 20 years’ experience of designing and building web pages, I learned to unlearn, to take some distances, to fight and also accept to lose for the better.
---

The web is so simple yet we often create cumbersome solutions. When the Developer Experience is balanced over User Experience, we may lose what's valuable in the process. This talk will try to look at my brief web history to find patterns and loops.

Building a more resilient web, understanding the value of the web and how we build a more sustainable heritage. I consider myself a web dinosaur, let's not condemn ourself to quickly to extinction.

My name is Marc Friederich, I would like you to read this article about the web I love. Still alive, a web [simple](#simple)<a>,</a> [accessible](#accessible), [resilient](#resilient), [fast](#fast), [distributed](#distributed), [diverse](#diverse), [empowering](#empower), [open](#open), [evolving](#evolving) and [playful](#playful). Through my 20 years’ experience of designing and building web pages, I learned to unlearn, to take some distances, to fight and also accept to lose for the better.

* * *

## Act 1 - We love a <a name="simple">simple</a> web

### First came the [hyperlink](https://en.wikipedia.org/wiki/Hyperlink#HTML)

The first interactive screen I built was on Hypercard. It was a quizz game about Expo.01 (The swiss national exposition which eventually became Expo02) for my classmates. I ended up building the quizz in HTML as it was easier to distribute.

Hypercard was a largely inspiring piece of software for Tim Berners-Lee to create the web. Hypercard is a proprietary software licensed by Apple. Even though [Ted Nelson first came with the hypertext concept](http://info.cern.ch/hypertext/WWW/Xanadu.html#Nelson) and Hypercard took it further the web really popularised it. History is trying to give us a lesson here.

Link was designed as not bidirectional, you refer to someone, but this someone doesn't have to know accept or reject, nor has anything to say about this relationship.

When you think about it, this is so much a missing opportunity, so poor compared to relational databases, proper objects linking. Yet I think this is precisely why web won over other technologies.

### Looking at proprietary solutions versus open web technologies

If you think of Macromedia Flash technologies, and before that Java Applets, they tried to fill a missing gap of web features. Web enthusiasts try hard to push web standards and browsers implement those gaps.

Looking at books referring flash sites in 2005 and realising how close to web experiences, we are now with CSS animations and Javascript. Yet to come with greater power with Houdini and full implementation of the Web Animation API. Not to mention WebAssembly allowing us to imagine an even more powerful web.

* * *

## Act 2. - We love an <a name="accessible">accessible</a> web

### Accessibility in all ways

You know accessibility matters. I mean accessibility for everyone to access information and features. Anyone across the globe and in the deep space. Poor connexions, old computers, without Javascript enabled, IE6 and yes Wget

### Show source

I largely learned by viewing the source of existing websites. Even though I understand CSS preprocessors and JavaScript compiler I feel really sad for the new generation. Open by default is an essential pillar to the web I love.

### Start with one file

A friend of mine (with an IT background but now in the chemistry industry) asked me what technologies he should use to build a new web service. To try to prototype an app idea he had last weekend.

> ME : Of course, you have this Javascript ecosystem or maybe with this language, going with a framework will be a safer choice maybe.
>
> HE : Ok, let's try this!
>
> ME : First install a package manager, to install another package manager, ...
>
> ...5 hours later... HE : What happen to the web?!

I know web development has evolved so much and we have tools to create safer and more robust solutions. Right but we need to find ways to increase accessibility to publish content, and publish app.

### Open knowledge

I feel hope, of course, with all information, courses and resources for beginners. Github offers a lot to newcomers who'd like to try and build web stuff. I will also mention later Beakerbrowser who vision a web without a server also to allow anyone to easily publish on the web. Accessibility to consume but also accessibility to publish is key to the web I love.

### Forgiveness by default

Part of this accessibility is the forgiveness of the browser. You may not close your tag or misspell a colour name but the browser will try is best to understand and interpret. But if it failed, it will ignore and forgive.

> Forgiveness by default is absolutely required for the kind of large-scale, worldwide adoption that the web enjoys.
>
> <footer>Jeff Atwood — <cite>JavaScript and HTML: Forgiveness by Default</cite></footer>

* * *

## Act 3. — We love a <a name="resilient">resilient</a> web

### Still working for the past

My mother's smartphone is an iPhone 3Gs. I know right, it sounds crazy not throwing a phone after 6 years. She asked me last week to fix the weather application as it didn't show the weather anymore. It appears that Yahoo weather API used by native Apple's iOS6 weather app is not supported anymore.

> "No problem mum, resilient web to the rescue!"

Another argument to the winning open web platform. **No**. Weather websites just crash safari and there are no other browser choices on the App Store.

### A rejected promise

I hate obsolescence, and I love theoretical web reslience. HTML, CSS and for a small part Javascript at their core have this rule :

*   1996 browser can display today's webpage
*   Today's browser can display a 1996 website

Of course we broke this rule, it is so tempting to use this huge file of Javascript to scrolljack my page. Let's try our best to avoid those situations. And just take acknowledgement and write down when we cross this line. It is so simple to have a no script tag explaining why we need JavaScript in the first place.

> You need JavaScript to access this content because we think our user needs to all buy a new smartphone each year.

I mean it. Try to really explain why Javascript is required here. You may also give another way of accessing the content.

* * *

## Act 4. – We all love a <a name="fast">fast</a> web

### So fast we forget data has to travel the world

There are tones of performance optimisation to consider when building on the web. Images, Code, Server caching technic, etc.

> Today's Solutions Are Tomorrow's Problems

HTTP/1.0 all in one CSS file for all your website optimisations are not as fast as HTTP/2.0 having a set of css files for every page types. No silver bullet. We need to test, learn and unlearn when a new implementation come.

### Eventually data is already here locally

I enjoy reading and testing offline first solutions. You may have heard of Hoodie which provides an offline first noBackend approach. Offline approach is a great step toward faster web. A more subtle approach to server-client strict model. Service Workers API is now so widely supported there is no excuse to invest time in learning how it works and applies to any website.

I cannot recommend enough Jermey Keith's new book "Going Offline". Dive in Service Workers API and explaining man-in-the-middle attacks and strategy to make your website faster for your user. PWA - Progressive Web Apps concept containing https, ServiceWorkers and the manifest is not about apps only. Every website will benefit from this.

There will be abstraction layers to service workers to ease developer life. As well explained in this book, every website (designwise, contentwise, architecturewise) will implement a different proxy strategy plus the API is already abstracted enough not to use a library.

* * *

## Act 5. – We all love a <a name="distributed">distributed</a> web

### We lost ground in the distributed web

Today Everything is a SaaS third party scripts loading on your webpage. Main web actors web presence is everywhere and when it failed, the Internet is down.

> Amazon admits that a typo took the internet down this week

This is not true, the Internet and the web are distributed technologies. Those dependencies should not be the norm. Just question whether it is worth the risk to relay on other providers. Feeding web monsters with more data will not help us have the distributed web we love.

### We just have to take power back

With initiatives like beakerborwser and dat:// protocol, there is an answer coming to a centralised web. Sir Tim Berners-Lee's new project Solid also gives a promising view of decentralised web and data ownership.

* * *

## Act 6. – We love a <a name="diverse">diverse</a> web

### Browser dominance is bad

I proudly displayed this animated gif on my website in 1999, god I was so wrong. We lived a long winter of innovation in web standards and web don't want to repeat history.

### Browser war is not new

> For perspective, by the end of 1992 there were **seven web browsers** which allowed users to surf the vast ocean of what was at the time only **22 known websites**.

It is our responsibility to install and test our work on other browsers (I know there are now just [6 different browser engines](https://en.wikipedia.org/wiki/Comparison_of_browser_engines)


### Get away from monopoly

There is a growing indie web movement praising for open and standard technologies. Helping the web stay as diverse as we can. Web mentions and ActivityPub are examples of open standards allowing a more diverse web allowing social media platforms to interact with each other.

> 1999: there are millions of websites all hyperlinked together  
> 2019: there are four websites, each filled with screenshots of the other three.
>
> <footer>David Masad @badnetworker</footer>

* * *

</a>

## Act 7. – We love an <a name="empower">empowering</a> people web

### Popups were so great

A version of my personal website was built over the concept of popup, you can define size and position with a little line of Javascript. From a user perspective, popup was just ads popping up when she exits a webpage. So browsers started providing popup blockers by default.

### It's a perspective question

Marketing people just closed amazing web technologies just by abusing it. Or people get power back by choosing to use the browser with popup blockers.

Now I may choose the next browser with native ad blocker technology or accepting cookies feature and so marketing has to be creative again.

* * *

## Act 8. – We love an <a name="open">open</a> web

### RSS as an exemple

RSS (Really Simple Syndication) is another example of simple web standards winning slowly over time. You may think RSS died with Google Reader, or social media took place as an intermediate syndication platform. Podcasts use RSS standards for instance. Google trends show a clear decline of the term "RSS" but the reality is more subtle.

> For a while, before a third of the planet had signed up for Facebook, RSS was simply how many people stayed abreast of news on the internet.
>
> <footer>Aaron Swartz</footer>

### Designed to embrace new usages

The way RSS was designed (read [a full story](https://twobithistory.org/2018/12/18/rss.html) to understand its origins and forks) shows how open standards allow innovations to grow.

* * *

## Act 9. – We love an <a name="evolving">evolving</a> web

### Without reloading the page

Framesets were so good. They were responsive by design, gave better performances to a website by loading only the content requested and also fun to play with. I love that we accepted to free the screen space and embrace a full scrolling experience at least for a while. I think both Fixed CSS positioning and SEO nearly killed framesets and then mobiles came and gave the _coup de grace_.

### Let the web evolve

Although it is nice to remember frames, but nostalgia doesn't really work when we think of the web. Layout systems evolve with usage but content like a liquid element takes his place in the medium.

* * *

## Act 10. – We love a <a name="playful">playful</a> web

### A JavaScript library to round corners

In 2006, I used some JavaScript to round corners of a div. It was really modern to use round corners because they were not possible to do (without tricks). I am not ashamed, it was totally unobstructive and progressive enhancement in mind. No JavaScript, you get square corners. This is still a playful web nowadays. Trying to push the boundaries a bit.

### Enthusiasm drives me

Call it our enthusiasm. But we can't wait to use a new web feature. We don't want to wait for a complete acceptance from browser vendors to start playing around. When we get a full acceptance of border-radius (IE9 in 2011) all the web became flat and squared design... At this point progressive enhancement became acceptable for our clients.

* * *

## Certitudes out of the way, but always keep web core values

In order to continue playing, we have to always reinvent the way we build websites. This empty field ready for us to explore. Some new technic seems really like a bad idea, but let's see where it will lead us. Just don't forget what we are trying to solve and to whom it benefits.
