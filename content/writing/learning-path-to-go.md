---
title: "My learning path to go"
date: 2018-09-22T16:14:31+02:00
showDate: true
draft: true
tags: ["blog","dev","go"]
description : Trying to dive into Go language for my next web project. This article details ressources and links.
---
## Why go ?
Convinced by Hugo (which run this website ;-)
https://grisha.org/blog/2017/04/27/simplistic-go-web-app/

## What I learnt
Learn by doing first with simple exemples explaining core concepts https://gowebexamples.com/. Simply this Hello World show the power of GoLang.

Having a ```main.go``` file with this content
````
package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)
	})

	http.ListenAndServe(":3000", nil)
}
````
execute ```$ go run main.go``` and you have a webserver, listening to localhost:3000 handling your request ;-)

## Manage dependencies is part of the language ...
```
$ go get -u github.com/gorilla/mux
```
To install mux http request handler ... and simply ```import ( "github.com/gorilla/mux" )``` to use it


https://gobyexample.com/


Projet exemple https://blue-jay.github.io/
https://www.sohamkamani.com/blog/2017/09/13/how-to-build-a-web-application-in-golang/
https://grisha.org/blog/2017/04/27/simplistic-go-web-app/

##  Then
- Choose a database


## Find a project

###

## Sources ...
