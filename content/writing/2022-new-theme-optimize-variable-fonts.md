---
title: "Here is a new design to my website - variable is the keyword"
date: 2022-11-11T18:05:59+01:00
showDate: true
draft: false
tags: ["blog","dev"]
description : With new colors reacting responsively and a new variable font
---

Using [Work Sans](https://github.com/weiweihuanghuang/Work-Sans) variable font to choose ... 

<h3 class="youpi">whatever weight I want</h3>

<style>
	.youpi{
		animation-duration: 2s;
	    animation-name: getbold;
		animation-iteration-count: infinite;
		animation-timing-function: ease-in-out;
		text-align: center;
		padding: 1em;
	}
	@keyframes getbold {
	  from {
		font-weight:1;
	  }
	  50%{
		font-weight: 999;
	  }
	  to {
		  font-weight:1;
	  }
	}
</style>

This is really nice to adapt weight to very use case. Just like this :

---

# The Story of Typewriters is a title1

## In the Country of the Last Things is a title2

### The Swiss Team has won the Solar Decathlon competition would mean 3

#### The strength of the Swiss team consists in its multidisciplinary cohesion is 4

##### This project also provided the students with a unique, unforgettable, and inspirational learning opportunity is 5

You can use the mark tag to <mark>highlight</mark> text.
  
<del>This line of text</del> is meant to be treated as deleted text.
  
<u>This line of text will render as underlined</u>
  
<small>This line of text is meant to be treated as fine print.</small>
  
<strong>This line rendered as bold text.</strong>
  
<em>This line rendered as italicized text.</em>

---

## A matter of footprint

Using a variable font with 900 different weights allow to have only 1 file loaded. 
But the original TTF is 300ko.
After a special treatment, the woff2 is only 30ko.

### How to compress a variable TTF font

installing Python, pip etc ...
```shell
sudo port install python39
sudo port select --set python python39
```

Install [fonttools](https://fonttools.readthedocs.io/en/latest/index.html) to manipulate font files
```shell
pip install fonttools
pip install brotli
```

Remove all glyphs except basic latin characters (punctuation, digits, letters)
```shell
fonttools subset WorkSans\[wght\].ttf  --output-file=worksans.ttf --unicodes=U+0020-007E
```

Use [woff2 library](https://github.com/google/woff2) to convert TTF to WOFF2 or using fonttools in only one line
```shell
fonttools subset WorkSans\[wght\].ttf  --output-file=worksans.woff --flavor=woff  --unicodes=U+0020-007E
```

I am really happy with Work Sans

## Variable color scheme 

My CSS goes with variable constructed around HSL attributes 
```CSS
:root {
	--color-h: 225; /* Hue General */
	
	/* background */ 
	--bg-color-s: 80%; /* Saturation */
	--bg-color-l: 96%; /* Lightness  */
	
	--bg-color: hsl(
  	var(--color-h), 
  	var(--bg-color-s), 
  	var(--bg-color-l)
	);
}
```

Then with a little unobstructive Javascript
```javascript
function gethue(){
  const x = 1 - (window.innerWidth) / 10000;
  const actual_color_hue = getComputedStyle(document.documentElement).getPropertyValue('--color-h');
  document.documentElement.style.setProperty('--color-h', Math.round(300*x));		
}
gethue();
window.addEventListener('resize', () => {
	gethue();
})
``` 

Just try resizing the page to see it in action.

Or simply drag the following input

<script>function getlivehue(){
   const x = 1 - (document.getElementById('hue').value) / 10000;
   const actual_color_hue = getComputedStyle(document.documentElement).getPropertyValue('--color-h');
   document.documentElement.style.setProperty('--color-h', Math.round(300*x));		
}</script>

<input type="range" id="hue" name="hue"
 min="0" max="5000" onmousemove="getlivehue();" style="width: 90%;">
 