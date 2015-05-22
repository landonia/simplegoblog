// Copyright 2013 Landon Wainwright. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Contains the structures and functions for the web server
package blog

import (
	"github.com/landonia/tollbooth"
	"github.com/landonia/tollbooth/config"
	"html/template"
	"log"
	"net/http"
)

// The data that is passed to all templates
type PageContent struct {
	Title       string
	Description string
	Posts       []*Post
	Post        *Post
}

// Adds a custom handler to the existing handlers
// Will not allow you to overwrite the existing blog paths
func (this *Blog) AddCustomHandler(path string, handler func(http.ResponseWriter, *http.Request), throttleLimit *config.Limiter) {

	// Add the custom handler
	http.Handle(path, tollbooth.LimitFuncHandler(throttleLimit, handler))
}

// Will start the blog on the chosen address
func (this *Blog) Start(addr string) error {

	// Read in all the post files
	err := this.loadPosts()
	if err != nil {
		return err
	}

	// Use tollbooth as a throttle limiter based on standard request IP. The limit will be for a second
	var throttleLimit = tollbooth.NewLimiter(this.configuration.RequestHandlerLimit.Max, this.configuration.RequestHandlerLimit.Ttl)

	// Setup the templates
	//this.templates = spitz.New(templatesdir, this.developmentMode)
	this.templates = template.Must(template.ParseFiles(this.getTemplatePath("header.html"),
		this.getTemplatePath("footer.html"), this.getTemplatePath("home.html"),
		this.getTemplatePath("post.html"), this.getTemplatePath("posts.html"),
		this.getTemplatePath("notfound.html"), this.getTemplatePath("about.html")))

	// Setup the handlers
	http.Handle("/", generateHandler(this, "home.html", viewHomeHandler, throttleLimit))
	http.Handle("/posts", generateHandler(this, "posts.html", viewPostsHandler, throttleLimit))
	http.Handle("/posts/", generateHandler(this, "post.html", viewPostHandler, throttleLimit))
	http.Handle("/about", generateHandler(this, "about.html", viewPostsHandler, throttleLimit))
	http.Handle("/notfound", generateHandler(this, "notfound.html", notFoundHandler, throttleLimit))

	// Add the file server for the asset directory
	http.Handle("/assets/", tollbooth.LimitHandler(throttleLimit,
		http.StripPrefix("/assets/", http.FileServer(http.Dir(this.configuration.Assetsdir)))))

	// Start the server
	log.Printf("Starting server using address: %s", addr)
	return http.ListenAndServe(addr, nil)
}

// Will generate a handler passing the current blog handler
func generateHandler(blog *Blog, template string, handler func(http.ResponseWriter, *http.Request, *Blog, string), throttleLimit *config.Limiter) http.Handler {

	// Just call the underlying function using the throttle middleware
	return tollbooth.LimitFuncHandler(throttleLimit, func(w http.ResponseWriter, r *http.Request) { handler(w, r, blog, template) })
}

// Handles all the requests to the home page
// This could be treated as just yet another post but its so easy to separate it makes sense to.
func viewHomeHandler(w http.ResponseWriter, r *http.Request, blog *Blog, template string) {

	// Only serve the home page if the path is /
	if r.URL.Path != "/" {

		// Redirect to the not found page
		http.Redirect(w, r, "/notfound", http.StatusFound)
		return
	}

	// We want to display the last n (cnfiguration) number of posts on the home page (if there are that many)
	recentPosts := blog.posts[:len(blog.posts)]
	if len(recentPosts) > blog.configuration.NoOfRecentPosts {
		recentPosts = recentPosts[:blog.configuration.NoOfRecentPosts]
	}

	blog.RenderTemplate(w, template, PageContent{Title: blog.configuration.Title, Posts: recentPosts})
}

// Handles all the requests to the posts page
func viewPostsHandler(w http.ResponseWriter, r *http.Request, blog *Blog, template string) {

	// Just send all the posts
	blog.RenderTemplate(w, template, PageContent{Title: blog.configuration.Title, Posts: blog.posts})
}

// handles all the requests for displaying a specific post
func viewPostHandler(w http.ResponseWriter, r *http.Request, blog *Blog, template string) {

	// Extract the post name
	postName := r.URL.Path[len("/posts/"):]

	// Locate the post
	post := blog.postMap[postName]
	if post == nil {

		// Redirect to the not found page
		http.Redirect(w, r, "/notfound", http.StatusFound)
		return
	}
	blog.RenderTemplate(w, template, PageContent{Title: post.Title, Post: post})
}

// Will be called when the requested page cannot be located
func notFoundHandler(w http.ResponseWriter, r *http.Request, blog *Blog, template string) {

	// Just send back the not found immediately
	w.WriteHeader(http.StatusNotFound)

	// Render the not found page
	blog.RenderTemplate(w, template, PageContent{Title: "Page Not Found"})
}

// Will render the chosen template
func (this *Blog) RenderTemplate(w http.ResponseWriter, tmpl string, data PageContent) {
	err := this.templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
