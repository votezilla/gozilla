// utils.go
package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"text/template" // Faster than "html/template", and less of a pain for safeHTML	
)

///////////////////////////////////////////////////////////////////////////////
//
// utility functions
//
///////////////////////////////////////////////////////////////////////////////
func assert(ok bool) {
    if !ok {
        panic("Assert failed!")
    }
}

func check(err error) {
    if err != nil {
        panic(err)
    }
}

func print(text string) {
	log.Println(text)
}

func printVal(label string, v interface{}) {
	log.Printf("%s: %v", label, v)
}

func printValX(label string, v interface{}) {
	log.Printf("%s: %x", label, v)
}


///////////////////////////////////////////////////////////////////////////////
//
// render template files
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
	log.Println(1)

	T := func(page string) string {
		return "templates/" + page + ".html"
	}

	templates = make(map[string]*template.Template)
	
	// HTML templates
	templates["form"]			= template.Must(template.ParseFiles(T("base"), T("form")))
	templates["frontPage"]		= template.Must(template.ParseFiles(T("base"), T("frontPage")))
	templates["news"]			= template.Must(template.ParseFiles(T("base"), T("news")))
	templates["newsSources"]	= template.Must(template.ParseFiles(T("base"), T("newsSources")))
	
	// Javascript snippets
	templates["registerDetailsScript"]	= template.Must(template.ParseFiles(T("registerDetailsScript")))
}

func executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	//log.Printf("executeTemplate: " + templateName)
	
	if flags.debug != "" {
		parseTemplateFiles()
	}

	err := templates[templateName].Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writes to io.Writer instead of http.ResponseWriter
func renderTemplate(w io.Writer, templateName string, data interface{}) {
	//log.Printf("renderTemplate: " + templateName)
	
	if flags.debug != "" {
		parseTemplateFiles()
	}

	err := templates[templateName].Execute(w, data)
	check(err)
}

// Render the table form, return the HTML string
func getFormHtml(tableForm TableForm) string {
	var formHTML bytes.Buffer
	renderTemplate(&formHTML, "tableForm", tableForm)
	return formHTML.String()
}