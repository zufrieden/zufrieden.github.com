---
title: "Clever Technique to Make It Shine with HDR Video in Background"
date: 2024-10-05T10:51:38+02:00
showDate: true
draft: false
tags: ["blog","dev"]
description : Using HDR's annoying brightness of videos rendering as a feature
---

## Using HDR's annoying brightness of videos rendering as a feature

If you’re on an HDR-capable device, you should see a bright ghost on your screen. Try lowering your device’s brightness, and you’ll notice that the ghost doesn’t seem to follow your setting. This might look like a bug at first glance, but let’s turn this unexpected behavior into a feature. The intense brightness can actually become a visually striking element when utilized correctly.





<style>
	
   @keyframes boooh {
  0%{
	  transform: scaleX(1);  
	  margin-left:0;
  }
  49%{
	margin-left:50vw;
  }
  49%{
	transform: scaleX(1);  
  }
  50%{
	transform: scaleX(-1);
  }
  99%{
	transform: scaleX(-1);  
  }
  100% {
    transform: scaleX(1);  
	margin-left:0;
  }
}	
	
#glowing{
	margin:30px 0;
	width: 160px;
	height: 160px;
	position: relative;
	animation-name: boooh;
	animation-duration: 12s;
	animation-iteration-count: infinite;
	animation-timing-function: ease-in-out;
}
#glowing > *{
	position: absolute;
	top:0;
	left:0;
	width: 160px;
	height: 160px;
}
#glowing video{
	object-fit: cover;
}
:root {
  --pixel-size: 10px;
  --body-color: transparent;
}

.ghost {
  display: grid;
  grid-template-columns: repeat(16, var(--pixel-size));
  grid-template-rows: repeat(16, var(--pixel-size));
/*   gap: 1px; */
}

.ghost div {
  background-color: var(--bg-color);
}
.ghost .body{
	background-color: var(--body-color);
}
.ghost .outline {
  background-color: color(display-p3 0 0 0 / 0.813);
}

.ghost .shadow {
  background-color: color(display-p3 1 0.94 0.94 / 0.673);
}

.ghost .arm {
  background-color:  color(display-p3 0 0 0 / 0.813);;
}

.ghost .left-eye {
  background-color:  color(display-p3 0 0 0 / 0.813);;
}

.ghost .right-eye {
  background-color:  color(display-p3 0 0 0 / 0.813);;
}

.ghost .mouth {
  background-color: color(display-p3 1 0.079 0 / 0.701);  
}

</style>
			
<div id="glowing">
<video preload="auto" video="" autoplay="" loop="" muted="" playsinline="" width="160px" height="160px">
<source src="/images/glowvideo.webm" type="video/webm">
<source src="/images/glowvideo.mp4" type="video/mp4">
</video>
<div class="ghost">
<!-- row 1 -->
<div></div>
<div></div>
<div></div>
<div></div>
<div></div>
<div></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="outline"></div>
<div></div>
<div></div>
<div></div>
<div></div>
<div></div>
<!-- row 2 -->
<div></div>
<div></div>
<div></div>
<div></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div class="outline"></div>
<div></div>
<div></div>
<div></div>
<!-- row 3 -->
<div></div>
<div></div>
<div></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div></div>
<div></div>
<!-- row 4 -->
<div></div>
<div></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div></div>
<!-- row 5 -->
<div></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="left-eye"></div>
<div class="body"></div>
<div class="right-eye"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div></div>
<!-- row 6 -->
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="arm"></div>
<div class="arm"></div>
<div class="arm"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="left-eye"></div>
<div class="body"></div>
<div class="right-eye"></div>
<div class="body"></div>
<div class="shadow"></div>
<div class="outline"></div>
<!-- row 7 -->
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="arm"></div>
<div class="body"></div>
<div class="body"></div>
<div class="arm"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="left-eye"></div>
<div class="body"></div>
<div class="right-eye"></div>
<div class="body"></div>
<div class="shadow"></div>
<div class="outline"></div>
<!-- row 8 -->
<div></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="arm"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="shadow"></div>
<div class="outline"></div>
<!-- row 9 -->
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="arm"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="shadow"></div>
<div class="outline"></div>
<!-- row 10 -->
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="shadow"></div>
<div class="outline"></div>
<!-- row 11 -->
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="mouth"></div>
<div class="shadow"></div>
<div class="outline"></div>
<!-- row 12 -->
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="body"></div>
<div class="mouth"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div></div>
<!-- row 13 -->
<div></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div></div>
<!-- row 14 -->
<div></div>
<div></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="body"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div></div>
<div></div>
<!-- row 15 -->
<div></div>
<div></div>
<div></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="shadow"></div>
<div class="outline"></div>
<div class="outline"></div>
<div></div>
<div></div>
<div></div>
<!-- row 16 -->
<div></div>
<div></div>
<div></div>
<div></div>
<div></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="outline"></div>
<div class="outline"></div>
<div></div>
<div></div>
<div></div>
<div></div>
<div></div>
</div>	
</div>

<!-- Took the ghost here https://codepen.io/dana-ciocan/pen/PoXVKrK -->

Creating a visually striking website or digital experience often comes down to the little tricks and techniques that give it that extra pop. By carefully layering other design elements over the video, you can create a dynamic, bright, and engaging experience for users.

## Source of this trick

With the excellent [newsletter from DataGif](https://datagif.fr/en/media-newsletter/), I discovered the [Digital Divinity](https://restofworld.org/series/digital-divinity/) long-read experience from *Rest of the World*. While reading [this article](https://restofworld.org/2024/divinity-influence-giacminhluat/), I noticed some magical and bright visual elements. I also use this the ghost from [CodePen](https://codepen.io/dana-ciocan/pen/PoXVKrK). They talk about the tricks [on their blog](https://restofworld.org/inside/digital-divinity-project/).
