// utils.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template" // Faster than "html/template", and less of a pain for safeHTML	
	"time"
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

///////////////////////////////////////////////////////////////////////////////
//
// logging
//
///////////////////////////////////////////////////////////////////////////////
type PrintMask uint
const (
	nw_		= PrintMask( 1)
	go_		= PrintMask( 2)
	sc_		= PrintMask( 4)
	db_ 	= PrintMask( 8)
	fo_		= PrintMask(16)
	po_		= PrintMask(32)
	
	all_	= PrintMask(64 - 1)
)

func print(text string) {
	log.Println(text)
}
func pr(mask PrintMask, text string) {
	if (mask & flags.printMask) != 0 {
		print(text)
	}
}

func printf(format string, args... interface{}) {
	log.Printf(format, args...)
}
func prf(mask PrintMask, format string, args... interface{}) {
	if (mask & flags.printMask) != 0 {
		printf(format, args...)
	}	
}

func printVal(label string, v interface{}) {
	log.Printf("%s: %#v", label, v)
}
func prVal(mask PrintMask, label string, v interface{}) {
	if (mask & flags.printMask) != 0 {
		printVal(label, v)
	}
}

func printValX(label string, v interface{}) {
	log.Printf("%s: %x", label, v)
}
func prValX(mask PrintMask, label string, v interface{}) {
	if (mask & flags.printMask) != 0 {
		printValX(label, v)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// math
//
///////////////////////////////////////////////////////////////////////////////
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

///////////////////////////////////////////////////////////////////////////////
//
// render template files
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
	T := func(page string) string {
		return "templates/" + page + ".html"
	}

	templates = make(map[string]*template.Template)
	
	// HTML templates
	templates["form"]			= template.Must(template.ParseFiles(T("base"), T("form")))
	templates["news"]			= template.Must(template.ParseFiles(T("base"), T("news")))
	templates["newsSources"]	= template.Must(template.ParseFiles(T("base"), T("newsSources")))
	templates["submit"]			= template.Must(template.ParseFiles(T("base"), T("submit")))
	
	// Javascript snippets
	templates["registerDetailsScript"]	= template.Must(template.ParseFiles(T("registerDetailsScript")))
}

func executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	//pr("executeTemplate: " + templateName)
	
	if flags.debug != "" {
		parseTemplateFiles()
	}

	err := templates[templateName].Execute(w, data)
	if err != nil {
		printVal("executeTemplate err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writes to io.Writer instead of http.ResponseWriter
func renderTemplate(w io.Writer, templateName string, data interface{}) {
	//pr("renderTemplate: " + templateName)
	
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

// Serves the specified HTML string as a webpage.
func serveHTML(w http.ResponseWriter, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, html)
}

// http.Get with a 'timeout'-second timeout.
func httpGet(url string, timeout float32) (*http.Response, error){
	var netClient = &http.Client{
	  Timeout: time.Duration(timeout) * time.Second,
	}
	return netClient.Get(url)
}

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	} 
	// Return the request as a string
	return strings.Join(request, "\n")
}