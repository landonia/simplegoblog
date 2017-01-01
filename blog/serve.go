// Copyright 2013 Landon Wainwright. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blog

import (
	"html/template"
	"net/http"

	"github.com/landonia/tollbooth"
	"github.com/landonia/tollbooth/config"
)

// PageContent data that is passed to all templates
type PageContent struct {
	Title       string
	Description string
	Posts       []*Post
	Post        *Post
}

// AddCustomHandler to the existing handlers
// Will not allow you to overwrite the existing blog paths
func AddCustomHandler(path string, handler func(http.ResponseWriter, *http.Request), throttleLimit *config.Limiter) {

	// Add the custom handler
	http.Handle(path, tollbooth.LimitFuncHandler(throttleLimit, handler))
}

// Start the blog on the chosen address
func (blog *Blog) Start(addr string) error {

	// Read in all the post files
	err := blog.loadPosts()
	if err != nil {
		return err
	}

	// Use tollbooth as a throttle limiter based on standard request IP. The limit will be for a second
	var throttleLimit = tollbooth.NewLimiter(blog.configuration.RequestHandlerLimit.Max, blog.configuration.RequestHandlerLimit.TTL)

	// Setup the templates
	//this.templates = spitz.New(templatesdir, this.developmentMode)
	blog.templates = template.Must(template.ParseFiles(blog.getTemplatePath("header.html"),
		blog.getTemplatePath("footer.html"), blog.getTemplatePath("home.html"),
		blog.getTemplatePath("post.html"), blog.getTemplatePath("posts.html"),
		blog.getTemplatePath("notfound.html"), blog.getTemplatePath("about.html")))

	// Setup the handlers
	http.Handle("/", generateHandler(blog, "home.html", viewHomeHandler, throttleLimit))
	http.Handle("/posts", generateHandler(blog, "posts.html", viewPostsHandler, throttleLimit))
	http.Handle("/posts/", generateHandler(blog, "post.html", viewPostHandler, throttleLimit))
	http.Handle("/about", generateHandler(blog, "about.html", viewPostsHandler, throttleLimit))
	http.Handle("/notfound", generateHandler(blog, "notfound.html", notFoundHandler, throttleLimit))

	// Add the file server for the asset directory
	http.Handle("/assets/", tollbooth.LimitHandler(throttleLimit,
		http.StripPrefix("/assets/", http.FileServer(http.Dir(blog.configuration.Assetsdir)))))

	// Start the server
	logger.Info("Starting server using address: %s", addr)
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

// RenderTemplate will render the chosen template
func (blog *Blog) RenderTemplate(w http.ResponseWriter, tmpl string, data PageContent) {
	err := blog.templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
