// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
)

type Page struct {
	Title string
	Body  []byte
}

var templates *template.Template = nil

// The templates for each HTML page to render.

// Page functions
func (p *Page) save() error {
	log.Printf("Page.save")
	
	filename := "posts/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

// Local functions
func loadPage(title string) (*Page, error) {
	log.Printf("loadPage %s", title)
	
	filename := "posts/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func topHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "", Body: []byte("")}
	renderTemplate(w, "top", p)
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	log.Printf("viewHandler")
	
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	log.Printf("editHandler")
	
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	log.Printf("saveHandler")
	
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	log.Printf("renderTemplate")
	
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func init() {
	log.Printf("init")
	
	var err error
	templates, err = template.ParseFiles("templates/top.html",
	                                     "templates/edit.html",
                                         "templates/view.html")
	if err != nil {
	    log.Fatal(err)
	}
}

func main() {
	log.Printf("main")
	
	http.HandleFunc("/", topHandler)
	
	log.Printf("main0");

	http.HandleFunc("/view/", makeHandler(viewHandler))
	
	log.Printf("main1")
	
	http.HandleFunc("/edit/", makeHandler(editHandler))
	
	log.Printf("main2")
	
	http.HandleFunc("/save/", makeHandler(saveHandler))

	log.Printf("main3")

	http.ListenAndServe(":8080", nil)
	
	log.Printf("main4")
}