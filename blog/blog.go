// Copyright 2013 Landon Wainwright. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package blog contains the base structures
package blog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/fsnotify.v1"
)

// The mutex for reading the posts in
var mutex = &sync.Mutex{}

// Event struct for event information
type Event struct {
	Op Op // File operation that triggered the event.
}

// Op describes a set of file operations.
type Op uint32

// These are the generalized file operations that can trigger a notification.
const (
	Update Op = 1 << iota
)

// ThrottleLimit defines the throttle limit for the blog
type ThrottleLimit struct {
	Max int64         // This is number of tokens allowed in the bucket
	TTL time.Duration // This is the time period that a token will be added to the bucket
}

// Configuration contains information such as file directories etc
type Configuration struct {
	DevelopmentMode     bool
	Postsdir            string
	Templatesdir        string
	Assetsdir           string
	Title               string
	NoOfRecentPosts     int
	RequestHandlerLimit ThrottleLimit
}

// Templates that are to be handled by this applicaton
type Templates struct {
	templates []string
}

// Blog is the root data store for this blog
type Blog struct {
	configuration *Configuration
	posts         Posts
	postMap       map[string]*Post
	templates     *template.Template
}

// Post is a representation of a single post within the blog
type Post struct {
	FileName string
	Created  time.Time
	Updated  time.Time
	Title    string
	Summary  string
	Body     string
}

// Posts type for an array of post pointers
type Posts []*Post

func (posts Posts) Len() int           { return len(posts) }
func (posts Posts) Swap(i, j int)      { posts[i], posts[j] = posts[j], posts[i] }
func (posts Posts) Less(i, j int) bool { return posts[i].Created.After(posts[j].Created) }

// SafeTitle will make the title safe for use within the URL
func (blog *Post) SafeTitle() string {

	// Replace all spaces of the title with '-'
	return strings.ToLower(strings.Replace(blog.Title, " ", "-", -1))
}

// SafeURL will make the title safe for use within the URL
func (blog *Post) SafeURL() string {

	// Now make URL safe
	return url.QueryEscape(blog.SafeTitle())
}

// BodySafe will return the body as HTML (as the html template will automatically escape it by default)
func (blog *Post) BodySafe() template.HTML {

	// Return an HTML element
	return template.HTML(blog.Body)
}

// New will create a new Blog serving content from the provided directory
func New(configuration *Configuration) *Blog {

	// New() allocates a new blog
	blog := &Blog{}
	log.Printf("Creating '%s' blog", configuration.Title)
	log.Printf("Loading posts from directory: %s", configuration.Postsdir)
	log.Printf("Loading templates from directory: %s", configuration.Templatesdir)
	log.Printf("Serving assets from directory: %s", configuration.Assetsdir)

	// Initialise the blog values
	return blog.init(configuration)
}

// init resets the blog data
func (blog *Blog) init(configuration *Configuration) *Blog {
	blog.configuration = configuration
	blog.posts = nil
	blog.postMap = make(map[string]*Post)

	// Set the number of recent posts if it has not been set
	if blog.configuration.NoOfRecentPosts == 0 {
		log.Println("Setting number of recent posts to default value of 3")
		blog.configuration.NoOfRecentPosts = 3
	}

	// // Set the default throttle limit
	if blog.configuration.RequestHandlerLimit.Max == 0 {
		log.Println("Setting request handler limit to default value of 1s")
		blog.configuration.RequestHandlerLimit = ThrottleLimit{Max: 10, TTL: time.Second}
	}

	// Add the watcher for the post directory
	updates := WatchPosts(configuration.Postsdir)

	// This is used to exit out of the current timer handlers
	timerExit := make(chan bool)

	// Start listening for the update events
	go func() {
		for {
			select {
			case event := <-updates:
				if event.Op == Update {

					// Cancel any existing timers
					select {
					case timerExit <- false:
					default:
					}

					// This is the function that wil be called when the timer is started
					go func(blog *Blog, exit chan bool) {

						// This will call the reload posts when the timer has ended or exit when the exit channel is called
						select {
						case <-time.NewTimer(time.Second * 10).C:

							// Now reload the posts
							log.Println("Post directory has changed")
							blog.loadPosts()
						case <-exit:
							// This will drop out of the block
						}
					}(blog, timerExit)
				}
			}
		}
	}()
	return blog
}

// Will return the path for the specific template name
func (blog *Blog) getTemplatePath(templateName string) string {

	// Return the path to the template
	return path.Join(blog.configuration.Templatesdir, templateName)
}

// Will read all the available posts from the file system
func (blog *Blog) loadPosts() error {
	mutex.Lock()
	defer mutex.Unlock()

	// Open the root application directory where the posts are stored
	// Read in each file and generate the post and tag objects
	fileInfos, err := ioutil.ReadDir(blog.configuration.Postsdir)
	if err != nil {
		log.Printf("Cannot read the files from %s", blog.configuration.Postsdir)
		return err
	}

	postsno := 0
	blog.postMap = make(map[string]*Post)
	log.Printf("Loading posts")
	for _, fi := range fileInfos {

		// Load the file (only .json files should be read)
		if filepath.Ext(fi.Name()) == ".json" {
			filePath := path.Join(blog.configuration.Postsdir, fi.Name())
			fi, err := os.Open(filePath)
			defer fi.Close()
			if err == nil {

				// Copy the file contents into the buffer
				var b bytes.Buffer
				_, err := b.ReadFrom(fi)
				if err == nil {

					// Create an empty post to copy the values into
					var post Post
					err := json.Unmarshal(b.Bytes(), &post)
					if err == nil {

						// Is there a post already with the same title?
						for blog.postMap[post.SafeTitle()] != nil {

							// Then we need to ensure that this post has a unique name
							post.Title = fmt.Sprintf("%s-", post.Title)
						}

						// Then the data was un-marshalled successfully and the post can be used
						postsno++
						post.FileName = fi.Name()
						blog.postMap[post.SafeTitle()] = &post
					}
				}
			}
		}
	}
	log.Printf("Finished loading %d posts", postsno)

	// Now sort the posts into the array
	newPosts := make([]*Post, postsno)
	i := 0
	for _, v := range blog.postMap {
		newPosts[i] = v
		i++
	}

	// Sort the array
	sort.Sort(Posts(newPosts))
	blog.posts = newPosts
	return nil
}

// WatchPosts will create a watcher of the directory
func WatchPosts(directory string) chan Event {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// Create the channel where events are pushed
	updates := make(chan Event)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {

					// Push the event onto the queue to get the system to update the posts
					updates <- Event{Op: Update}
				}
			}
		}
	}()

	// Attempt to watch the directory
	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}
	return updates
}
