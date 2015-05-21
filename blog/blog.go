// Copyright 2013 Landon Wainwright. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Contains the base structures
package blog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/fsnotify.v1"
	"html/template"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The base configuration for a new blog.
// The Configuration contains information such as file directories etc
type Configuration struct {
	DevelopmentMode bool
	Postsdir        string
	Templatesdir    string
	Assetsdir       string
	Title           string
}

// Contains the templates that are to be handled by this applicaton
type Templates struct {
	templates []string
}

// Blog is the root data store for this blog
type Blog struct {
	configuration *Configuration
	posts         []*Post
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

// This will make the title safe for use within the URL
func (this *Post) SafeTitle() string {

	// Replace all spaces of the title with '-'
	return strings.ToLower(strings.Replace(this.Title, " ", "-", -1))
}

// This will make the title safe for use within the URL
func (this *Post) SafeURL() string {

	// Now make URL safe
	return url.QueryEscape(this.SafeTitle())
}

// Will return the body as HTML (as the html template will automatically escape it by default)
func (this *Post) BodySafe() template.HTML {

	// Return an HTML element
	return template.HTML(this.Body)
}

// Sort functionality to sort the posts in order they were created
type ByCreated []*Post

func (this ByCreated) Len() int           { return len(this) }
func (this ByCreated) Swap(i, j int)      { this[i], this[j] = this[j], this[i] }
func (this ByCreated) Less(i, j int) bool { return this[i].Created.After(this[j].Created) }

// Will create a new Blog serving content from the provided directory
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

// Init resets the blog data
func (this *Blog) init(configuration *Configuration) *Blog {
	this.configuration = configuration
	this.posts = nil
	this.postMap = make(map[string]*Post)
	// Add the watcher
	this.ExampleNewWatcher(configuration.Postsdir)
	return this
}

// Will return the path for the specific template name
func (this *Blog) getTemplatePath(templateName string) string {

	// Return the path to the template
	return path.Join(this.configuration.Templatesdir, templateName)
}

// Will read all the available posts from the file system
func (this *Blog) loadPosts() error {

	// Open the root application directory where the posts are stored
	// Read in each file and generate the post and tag objects
	fileInfos, err := ioutil.ReadDir(this.configuration.Postsdir)
	if err != nil {
		log.Printf("Cannot read the files from %s", this.configuration.Postsdir)
		return err
	}

	postsno := 0
	this.postMap = make(map[string]*Post)
	log.Printf("Loading posts")
	for _, fi := range fileInfos {

		// Load the file (only .json files should be read)
		if filepath.Ext(fi.Name()) == ".json" {
			filePath := path.Join(this.configuration.Postsdir, fi.Name())
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
						for this.postMap[post.SafeTitle()] != nil {

							// Then we need to ensure that this post has a unique name
							post.Title = fmt.Sprintf("%s-", post.Title)
						}

						// Then the data was un-marshalled successfully and the post can be used
						postsno += 1
						post.FileName = fi.Name()
						this.postMap[post.SafeTitle()] = &post
					}
				}
			}
		}
	}
	log.Printf("Finished loading %d posts", postsno)

	// Now sort the posts into the array
	this.posts = make([]*Post, postsno)
	i := 0
	for _, v := range this.postMap {
		this.posts[i] = v
		i += 1
	}

	// Sort the array
	sort.Sort(ByCreated(this.posts))
	return nil
}

// This will write the post to disk
func (this *Blog) SavePost(post Post) error {

	// Add a created time stamp
	if &post.Created == nil {
		log.Println("Created a new time stamp for post")
		post.Created = time.Now()
	}

	// Update the updated time stamp
	post.Updated = time.Now()

	// Marshall this to disk
	b, err := json.Marshal(post)
	if err != nil {
		log.Println("Unable to marshall Post")
		return err
	}

	// Return if the bytes array is empty
	if len(b) == 0 {
		log.Println("The Post contains no content to write to disk")
		return errors.New("There is no content to write to disk")
	}

	// Write the file out to disk

	filePath := path.Join(this.configuration.Postsdir, fmt.Sprintf("%d.json", post.Created.Unix()))
	fo, err := os.Create(filePath)
	defer fo.Close()
	if err != nil {
		log.Println("Unable to create Post: %s", filePath)
		return err
	}

	// Write the bytes to disk
	var buffer bytes.Buffer
	_, err = buffer.Write(b)
	if err != nil {
		log.Println("Unable to write Post bytes to buffer")
		return err
	}
	_, err = buffer.WriteTo(fo)
	return err
}

// This will create a watcher of the directory
func (this *Blog) ExampleNewWatcher(directory string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)

					// The posts directory has changed so we need to do reload the posts
					this.loadPosts()
				}
			}
		}
	}()

	// Attempt to watch the directory
	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}
}
