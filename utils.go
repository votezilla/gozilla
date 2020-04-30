// utils.go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)


///////////////////////////////////////////////////////////////////////////////
//
// assertion functions
//
///////////////////////////////////////////////////////////////////////////////
func assert(ok bool) {
    if !ok {
        panic("Assert failed!")
    }
}

func assertMsg(ok bool, errorMsg string) {
    if !ok {
        panic(errorMsg)
    }
}

func check(err error) {
    if err != nil {
        panic(err)
    }
}

func nyi() { panic("Not yet implemented!") }


///////////////////////////////////////////////////////////////////////////////
//
// math constants
//
///////////////////////////////////////////////////////////////////////////////
const MaxInt   = int(^uint(0) >> 1)
const MaxInt64 = int64(^uint64(0) >> 1)


///////////////////////////////////////////////////////////////////////////////
//
// math functions
//
///////////////////////////////////////////////////////////////////////////////
func ternary_int(b bool, i int, j int) 			int 	{ if b { return i } else { return j } }
func ternary_int64(b bool, i int64, j int64)    int64 	{ if b { return i } else { return j } }
func ternary_uint64(b bool, i uint64, j uint64) uint64 	{ if b { return i } else { return j } }
func round(f float32) int { return int(f + .5) }
func min_int(i int, j int) int { return ternary_int(i < j, i, j) }
func max_int(i int, j int) int { return ternary_int(i > j, i, j) }
func getBitFlag(flags, mask uint64) bool { return (flags & mask) != 0 }

// Inline switch which takes and returns int values.
// e.g. switch_int(2, // switch value:
//			0, 100,   // case 0: return 100
//			1, 200,   // case 1: return 200
//			2, 300)   // case 2: return 300
//		returns 300
func switch_int(switch_val int, cases_and_values ...int) int {
	prVal("switch_val", switch_val)
	prVal("cases_and_values", cases_and_values)

	for c := 0; c + 1 < len(cases_and_values); c += 2 {
		if switch_val == cases_and_values[c] {
			return cases_and_values[c + 1]
		}
	}
	return -1; // This is the default (not found) flag value.
}


///////////////////////////////////////////////////////////////////////////////
//
// string functions
//
///////////////////////////////////////////////////////////////////////////////
func ternary_str(b bool, s1 string, s2 string) 	string 	{ if b { return s1 } else { return s2 } }
func bool_to_str(b bool) string { return ternary_str(b, "true", "false") }
func str_to_bool(s string) bool { return s == "true" }
func coalesce_str(s1 string, s2 string) string { if s1 != "" { return s1 } else { return s2 } }


///////////////////////////////////////////////////////////////////////////////
//
// array functions
//
///////////////////////////////////////////////////////////////////////////////

// Return true if item is in array.
func contains_int64(array []int64, item int64) bool {
	for _, a := range array {
		if a == item {
			return true
		}
	}
	return false
}


/////////////////////////////////////////////////////////////////////////
//
// logging
//
///////////////////////////////////////////////////////////////////////////////
func pr(text string) {
	log.Println(text)
}

func prf(format string, args... interface{}) {
	log.Printf(format, args...)
}

func prVal(label string, v interface{}) {
	log.Printf("%s: %#v", label, v)
}

func prValX(label string, v interface{}) {
	log.Printf("%s: %x", label, v)
}


///////////////////////////////////////////////////////////////////////////////
//
// render template files
//
///////////////////////////////////////////////////////////////////////////////
func executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	pr("executeTemplate: " + templateName)

	if flags.debug != "" {
		parseTemplateFiles()
	}

	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, but is necessary for rendering HTML, such as auto-generated forms.
	_, ok := htemplates[templateName]
	var err error
	if ok {
		err = htemplates[templateName].Execute(w, data)
	} else {
		err = ttemplates[templateName].Execute(w, data)
	}
	if err != nil {
		prVal("executeTemplate err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// writes to io.Writer instead of http.ResponseWriter
func renderTemplate(w io.Writer, templateName string, data interface{}) {
	pr("renderTemplate: " + templateName)

	if flags.debug != "" {
		parseTemplateFiles()
	}

	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, but is necessary for rendering HTML, such as auto-generated forms.
	_, ok := htemplates[templateName]
	var err error
	if ok {
		err = htemplates[templateName].Execute(w, data)
	} else {
		err = ttemplates[templateName].Execute(w, data)
	}
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
func httpGet_Old(url string, timeout float32) (*http.Response, error){
	var netClient = &http.Client{
	  Timeout: time.Duration(timeout) * time.Second,
	}
	return netClient.Get(url)
}


// http.Get with a 'timeout'-second timeout.
func httpGet(url string, timeout float32) (*http.Response, error){
	return httpGet_Old(url, timeout)
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

func parseUrlParam(r *http.Request, name string) string {
	values, ok := r.URL.Query()[name]

	if !ok || len(values) < 1 {
		return ""
	} else {
		return values[0]
	}
}
