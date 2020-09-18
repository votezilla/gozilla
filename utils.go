// utils.go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	timers = make(map[string]time.Time)
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

// HTML-spewing assertion functions:
func assertHtml(w http.ResponseWriter, ok bool) {
    if !ok {
        serveErrorMsg(w, "Assert failed!")
    }
}

func assertMsgHtml(w http.ResponseWriter, ok bool, errorMsg string) {
    if !ok {
        serveErrorMsg(w, errorMsg)
    }
}


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
func ternary_int    (b bool, i, j int) 		int 	{ if b { return i } else { return j } }
func ternary_int64  (b bool, i, j int64)    int64 	{ if b { return i } else { return j } }
func ternary_uint64 (b bool, i, j uint64) 	uint64 	{ if b { return i } else { return j } }
func ternary_float32(b bool, i, j float32)	float32 { if b { return i } else { return j } }
func round(f float32) 						int 	{ return int(f + .5) }
func min_int(i int, j int) 					int 	{ return ternary_int(i < j, i, j) }
func max_int(i int, j int) 					int 	{ return ternary_int(i > j, i, j) }
func getBitFlag(flags, mask uint64) 		bool 	{ return (flags & mask) != 0 }
func ceil_div(dividend int, divisor int) 	int 	{ return (dividend + (divisor - 1)) / divisor; }

// Inline switch which takes and returns int values.
// e.g. switch_int(2, // switch value:
//			0, 100,   // case 0: return 100
//			1, 200,   // case 1: return 200
//			2, 300)   // case 2: return 300
//		returns 300
func switch_int(switch_val int, cases_and_values ...int) int {
	//prVal("switch_val", switch_val)
	//prVal("cases_and_values", cases_and_values)

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
func str_to_int64(s string) int64 { i, err := strconv.ParseInt(s, 10, 64); check(err); return i }
func str_to_int(s string) int { i, err := strconv.Atoi(s); check(err); return i }
func int_to_str(i int) string { return strconv.Itoa(i) }

// Truncate and add "..." if text is longer than maxLength.
func ellipsify(s string, maxLength int) string {
	length := len(s)

	if length > maxLength {
		s = s[0:maxLength]
		s += "..."
	}

	return s
}

// Maps the input from an array of strings to an output array of strings, using the map function.
func map_str(mapFn func(string)string, input []string) []string {
	output := make([]string, len(input))
	for i := range(input) {
		output[i] = mapFn(input[i])
	}
	return output
}

// SECURITY_TODO: Note that in Go, map is unordered, so the replaces may happen in any order,
// But the double backslash has to be applied before any other rule. You might want to use [][2]string for the rules instead.
func sqlEscapeString(value string) string {
    replace := map[string]string{"\\":"\\\\", "'":`\'`, "\\0":"\\\\0", "\n":"\\n", "\r":"\\r", `"`:`\"`, "\x1a":"\\Z"} // "

    for b, a := range replace {
        value = strings.Replace(value, b, a, -1)
    }

    return value;
}


///////////////////////////////////////////////////////////////////////////////
//
// bool functions
//
///////////////////////////////////////////////////////////////////////////////
func ifthen(a, b bool) bool	{ return !a || (a && b) 	  }  // if then (math)
func iff(a, b bool)	   bool	{ return a && b || (!a && !b) }  // if and only if


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

///////////////////////////////////////////////////////////////////////////////
//
// time
//
///////////////////////////////////////////////////////////////////////////////
func getTimeSinceString(publishedAt time.Time, longform bool) string {
	timeSince 	:= time.Since(publishedAt)
	seconds 	:= timeSince.Seconds()
	minutes 	:= timeSince.Minutes()
	hours 		:= timeSince.Hours()
	days 		:= hours / 24.0
	weeks 		:= days / 7.0
	years 		:= days / 365.0

	if longform {
		s := ""
		if years > 20.0 {
			s = "a long time"
		} else if years >= 1.0 {
			s = strconv.FormatFloat(years, 'f', 0, 32) + " year" + ternary_str(years >= 2.0, "s", "")
		} else if weeks >= 1.0 {
			s = strconv.FormatFloat(weeks, 'f', 0, 32) + " week" + ternary_str(weeks >= 2.0, "s", "")
		} else if days >= 1.0 {
			s = strconv.FormatFloat(days, 'f', 0, 32) + " day" + ternary_str(days >= 2.0, "s", "")
		} else if hours >= 1.0 {
			s = strconv.FormatFloat(hours, 'f', 0, 32) + " hour" + ternary_str(hours >= 2.0, "s", "")
		} else if minutes >= 1.0 {
			s = strconv.FormatFloat(minutes, 'f', 0, 32) + " minute" + ternary_str(minutes >= 2.0, "s", "")
		} else {
			s = strconv.FormatFloat(seconds, 'f', 0, 32) + " second" + ternary_str(seconds >= 2.0, "s", "")
		}
		s += " ago"
		return s
	} else {  // Short form
		if years > 20.0 {
			return "old"
		} else if years >= 1.0 {
			return strconv.FormatFloat(years, 'f', 0, 32) + "y"
		} else if weeks >= 1.0 {
			return strconv.FormatFloat(weeks, 'f', 0, 32) + "w"
		} else if days >= 1.0 {
			return strconv.FormatFloat(days, 'f', 0, 32) + "d"
		} else if hours >= 1.0 {
			return strconv.FormatFloat(hours, 'f', 0, 32) + "h"
		} else if minutes >= 1.0 {
			return strconv.FormatFloat(minutes, 'f', 0, 32) + "m"
		} else {
			return strconv.FormatFloat(seconds, 'f', 0, 32) + "s"
		}
	}
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

// TODO: change prVal to prv
func prVal(label string, v interface{}) {
	log.Printf("%s: %#v", label, v)
}

func prx(label string, v interface{}) {
	log.Printf("%s: %x", label, v)
}

func prp(label string, v interface{}) {
	log.Printf("%s: %p", label, v)
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
	}// else {
	//	err = ttemplates[templateName].Execute(w, data)
	//}
	if err != nil {
		check(err)
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
	}// else {
	//	err = ttemplates[templateName].Execute(w, data)
	//}
	check(err)
}

// Serves the specified HTML string as a webpage.
func serveHTML(w http.ResponseWriter, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, html)
}

func serveErrorMsg(w http.ResponseWriter, errorMsg string) {
	prVal("Error: ", errorMsg)
	http.Error(w, errorMsg, http.StatusInternalServerError)
}

func serveError(w http.ResponseWriter, err error) {
	serveErrorMsg(w, err.Error())
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
		//name = strings.ToLower(name)
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

// Tutorial popup
func tutorialHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("tutorialHandler")

	userId, username := GetSessionInfo(w, r)

	executeTemplate(w, kTutorial, makeFrameArgs(r, "Tutorial", "", "tutorial", userId, username))
}

/////////////////////////////////////////////////////////////////////////
//
// timing / profiling
//
// This code works great!  Remember to comment out when not using to save performance.
//
///////////////////////////////////////////////////////////////////////////////
func startTimer(name string) {
//	timers[name] = time.Now()
}
func endTimer(name string) {
//	start, found := timers[name]
//	assert(found)
//	timeElapsed := time.Since(start)
//	prf("timeElapsed(%s): %2.3f", name, timeElapsed.Seconds())
}
