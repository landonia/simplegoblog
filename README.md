# simplegoblog

A simple blog system written using the Go Language.

## Overview

The simple go blog system uses only the base packages contained within Go for creating a blog.
All posts are saved and loaded from disk and cached in memory. This makes it extremely easy
to get started as the executable can be run directly on the required machine without any 
external requirements, such as a database.

To create new posts you simply have to create a new file using the template and save it to
the posts folder.

## Maturity

This is my first application written using the Go Language and therefore you should expect
some bugs. 

## Installation

With a healthy Go Language installed, simply run `go get github.com/landonia/simplegoblog/blog`

## Out of Box Example

My blog [landotube](https://github.com/landonia/landotube) was written using simplegoblog.

## Custom Example
    
	package main

	import (
		"flag"
		"github.com/landonia/simplegoblog/blog"
	)

	func main() {
	
		// Define flags
		var postsdir, templatesdir, assetsdir string
		flag.StringVar(&postsdir, "pdir", "../posts", "the directory for storing the posts")
		flag.StringVar(&templatesdir, "tdir", "../templates", "the directory containing the templates")
		flag.StringVar(&assetsdir, "adir", "../assets", "the directory containing the assets")
		flag.Parse()
	
		// Create a new configuration containing the info
		config := &blog.Configuration{DevelopmentMode:true, Postsdir:postsdir, Templatesdir:templatesdir, Assetsdir:assetsdir}
	
		// Create a new blog passing along the configuration
		b := blog.New(config)
	
		// Start the server
		err := b.Start(":8080")
		if err != nil {
			panic()
		}
	}
	
## Future

As the blog posts are marshalled to/from json and written to disk it would make sense
to add a feature that would allow you to use a JSON backed data store such as mongodb.

## About

simplegoblog was written by [Landon Wainwright](http://www.landotube.com) | [GitHub](https://github.com/landonia). 

Follow me on [Twitter @landotube](http://www.twitter.com/landotube)! Although I don't really tweet much tbh.
