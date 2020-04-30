// gozilla.go
package main

import (
	"fmt"
	"net/http"

	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, but is necessary for rendering HTML, such as auto-generated forms.
	htemplate "html/template"
	ttemplate "text/template"
)

var (
	htemplates  map[string]*htemplate.Template
	ttemplates  map[string]*ttemplate.Template

	err		 	error

	// NavMenu (constant)
	navMenu		= []string{"news", "create", "history"}
)


// Template arguments for webpage template.
type PageArgs struct {
	Title			string
	Script			string
}

const (
	kForm = "form"
	kArticle = "article"
	kNews = "news"
	kNewsSources = "newsSources"
	kCreate = "create"
	kCreateBlog = "createBlog"
	kCreateLink = "createLink"
	kCreatePoll = "createPoll"
	kViewPollResults = "viewPollResults"
	kRegisterDetailsScript = "registerDetailsScript"
)

///////////////////////////////////////////////////////////////////////////////
//
// TODO: get user's ip address
//       1) To log in the database when user is first created.
//		 2) To set their location in registerDetails and save them time.
// USING: https://play.golang.org/p/Z6ATIgo_IM
//        https://stackoverflow.com/questions/27234861/correct-way-of-getting-clients-ip-addresses-from-http-request-golang
//
// (WAIT TIL TESTING FROM AWS, OTHERWISE IT'S LOCALHOST, BASICALLY MEANINGLESS)
//
///////////////////////////////////////////////////////////////////////////////
func ipHandler(w http.ResponseWriter, r *http.Request) {
	remoteAddr	 := r.RemoteAddr
	forwardedFor := r.Header.Get("X-Forwarded-For")

	fmt.Fprintf(w, "<p>remote addr: %s</p>", remoteAddr)
	fmt.Fprintf(w, "<p>forwarded for: %s</p>", forwardedFor)
	fmt.Fprintf(w, "<br><p>r: %+v</p>", r)
}

///////////////////////////////////////////////////////////////////////////////
//
// handler wrapper - Each request should refresh the session.
//
///////////////////////////////////////////////////////////////////////////////
func hwrap(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	// TODO: we could add DNS Attack code defense here.  Check the ip, apply various masks.

	return func(w http.ResponseWriter, r *http.Request) {
		prf("\nHandling request from: %s\n", formatRequest(r))

		handler(w, r)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// parse template files - Establishes the template inheritance structure for Votezilla.
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
	T := func(page string) string {
		return "templates/" + page + ".html"
	}

	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, but is necessary for rendering HTML, such as auto-generated forms.
	htemplates = make(map[string]*htemplate.Template)
	ttemplates = make(map[string]*ttemplate.Template)

	// HTML templates
	ttemplates[kForm]		= ttemplate.Must(ttemplate.ParseFiles(T("base"), T("form"), T("defaultForm")))
	htemplates[kArticle]	= htemplate.Must(htemplate.ParseFiles(T("base"), T("frame"), T("article"), T("comments")))
	htemplates[kNews]		= htemplate.Must(htemplate.ParseFiles(T("base"), T("frame"), T("news")))
	htemplates[kNewsSources]= htemplate.Must(htemplate.ParseFiles(T("base"), T("newsSources")))
	htemplates[kCreate]		= htemplate.Must(htemplate.ParseFiles(T("base"), T("create")))
	ttemplates[kCreateBlog]	= ttemplate.Must(ttemplate.ParseFiles(T("base"), T("form"), T("createBlog")))
	ttemplates[kCreateLink]	= ttemplate.Must(ttemplate.ParseFiles(T("base"), T("form"), T("createLink")))
	ttemplates[kCreatePoll]	= ttemplate.Must(ttemplate.ParseFiles(T("base"), T("createPoll")))

	// Popup forms (they do not inherit from 'base')
	htemplates[kViewPollResults]= htemplate.Must(htemplate.ParseFiles(T("viewPollResults"), T("comments")))

	// Javascript snippets
	ttemplates[kRegisterDetailsScript]	= ttemplate.Must(ttemplate.ParseFiles(T("registerDetailsScript")))
}

///////////////////////////////////////////////////////////////////////////////
//
// program entry
//
///////////////////////////////////////////////////////////////////////////////
func init() {
	print("init")

	parseTemplateFiles()
}

func WebServer() {
	InitSecurity()

	http.HandleFunc("/",                		hwrap(newsHandler))
	http.HandleFunc("/ajaxCreateComment/",		hwrap(ajaxCreateComment))
	http.HandleFunc("/ajaxPollVote/",			hwrap(ajaxPollVoteHandler))
	http.HandleFunc("/ajaxScrapeImageURLs/",	hwrap(ajaxScrapeImageURLs))
	http.HandleFunc("/ajaxVote/",				hwrap(ajaxVoteHandler))
	http.HandleFunc("/article/",       			hwrap(articleHandler))
	http.HandleFunc("/create/",   				hwrap(createHandler))
	http.HandleFunc("/createBlog/",   			hwrap(createBlogHandler))
	http.HandleFunc("/createLink/",   			hwrap(createLinkHandler))
	http.HandleFunc("/createPoll/",   			hwrap(createPollHandler))
	http.HandleFunc("/forgotPassword/", 		hwrap(forgotPasswordHandler))
	http.HandleFunc("/history/",        		hwrap(historyHandler))
	http.HandleFunc("/ip/",             		hwrap(ipHandler))
	http.HandleFunc("/login/",          		hwrap(loginHandler))
	http.HandleFunc("/logout/",         		hwrap(logoutHandler))
	http.HandleFunc("/news/",           		hwrap(newsHandler))
	http.HandleFunc("/register/",       		hwrap(registerHandler))
	http.HandleFunc("/registerDetails/",		hwrap(registerDetailsHandler))
//	http.HandleFunc("/registerDone/",   		hwrap(registerDoneHandler))     // being called directly from registerDetailsHandler
	http.HandleFunc("/viewPollResults/",   		hwrap(viewPollResultsHandler))

	// Server static file.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Special handling for favicon.ico.
	http.Handle("/favicon.ico", http.FileServer(http.Dir("./static")))

	pr("Listening on http://localhost:" + flags.port + "...")
	http.ListenAndServe(":" + flags.port, nil)
}

func main() {
	print("main")

	parseCommandLineFlags()

	OpenDatabase()
	defer CloseDatabase()

	if flags.imageServer != "" {
		ImageServer()
	} else if flags.newsServer != "" {
		NewsServer()
	} else {
		WebServer()
	}
}


