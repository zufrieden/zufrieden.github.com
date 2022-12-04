---
title: "Moving out of Twitter and getting my content back"
date: 2022-12-04T14:27:09+01:00
showDate: true
draft: true
tags: ["blog","dev"]
description : Exporting my user data and converting it here
---

- Exporting my twitter data
- Using this code https://github.com/kidsil/hugo-data-to-pages
- Modifications to convert dates
- Rename files

## YAML export of every tweet, retweet

```yaml
 - tweet:
  edit_info:
	initial:
	  editTweetIds:
		- '1594349054363721728'
	  editableUntil: '2022-11-20T15:46:50.790Z'
	  editsRemaining: '5'
	  isEditEligible: false
  retweeted: false
  source: '<a href="http://tapbots.com/tweetbot" rel="nofollow">Tweetbot for iΟS</a>'
  entities:
	hashtags: []
	symbols: []
	user_mentions:
	  - name: Janna bastow@mastodon.social
		screen_name: simplybastow
		indices:
		  - '3'
		  - '16'
		id_str: '16231420'
		id: '16231420'
	urls: []
  display_text_range:
	- '0'
	- '67'
  favorite_count: '0'
  id_str: '1594349054363721728'
  truncated: false
  retweet_count: '0'
  id: '1594349054363721728'
  created_at: 'Sun Nov 20 15:16:50 +0000 2022'
  favorited: false
  full_text: 'RT @simplybastow: I mean, MySpace and ICQ are still technically up.'
  lang: en
  ```

## Converting yaml to markdown files
```JavaScript
const regex = /(^|[^@\w])@(\w{1,15})\b/g;
const replace = '$1<a href=\'http://twitter.com/$2\'>@$2</a>';
let tweetDate = new Date(pages[j].tweet.created_at);
let tweetfrontmatter = '---'+ '\n' 
	+'date: '+tweetDate.toISOString()+'\n'
	+'title: "'+pages[j].tweet.full_text.replace( regex, replace )+'″"\n'
	+'user_mentions: "'+pages[j].tweet.entities.user_mentions.name+'"\n'
	+'tweet_id: "'+pages[j].tweet.id+'"\n'
	+'pub_type: "tweet"\n'
	+'---'+'\n';
const pagePath = config.root + config.contentPath + '/' + formatDate(tweetDate) + '_' + pages[j].tweet.id;
```