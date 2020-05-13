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
	kArticle = "article"
	kCreate = "create"
	kCreateBlog = "createBlog"
	kCreateLink = "createLink"
	kCreatePoll = "createPoll"
	//kForm = "form"
	kLogin = "login"
	kLoginPopup = "loginPopup"
	kNews = "news"
	kNewsSources = "newsSources"
	kNuForm = "nuForm"
	kNuFormPopup = "nuFormPopup"
	kRegisterDetails = "registerDetails"
	kRegisterDetailsNopopup = "registerDetailsNopopup"
	//kRegisterDetailsScript = "registerDetailsScript"
	kRegister = "register"
	kRegisterPopup = "register"
	kTestPopup = "testPopup"
	kViewPollResults = "viewPollResults"
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
// parse template files - Establishes the template inheritance structure for Votezilla HTML code.
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, but is necessary for rendering HTML, such as auto-generated forms.
	htemplates = make(map[string]*htemplate.Template)
	ttemplates = make(map[string]*ttemplate.Template)

	getTemplatePath := func(page string) string {
		return "templates/" + page + ".html"
	}
	// We're trying to just use hDefineTemplate, since it prevents against HTML injection.
	//   Templates make it possible to use hDefineTemplate.
	//tDefineTemplate := func(handle string, filenames ...string) {
	//	ttemplates[handle] = ttemplate.Must(ttemplate.ParseFiles(map_str(getTemplatePath, filenames)...))
	//}
	hDefineTemplate := func(handle string, filenames ...string) {
		htemplates[handle] = htemplate.Must(htemplate.ParseFiles(map_str(getTemplatePath, filenames)...))
	}

	// HTML templates
	//tDefineTemplate(kForm, 			"base", "narrow", "frame", "form", "defaultForm")
	hDefineTemplate(kNuForm, 		"base", "narrow", "frame", "nuField", "nuForm", "defaultForm")
	hDefineTemplate(kArticle, 		"base", "wide", "frame", "article", "comments")
	hDefineTemplate(kNews, 			"base", "wide", "frame", "news")
	hDefineTemplate(kNewsSources,	"base", "wide", "frame", "newsSources")  // nyi

	hDefineTemplate(kCreate, 		"base", "narrow", "minFrame", "nuField", "create")
	hDefineTemplate(kCreateBlog, 	"base", "narrow", "minFrame", "nuField", "createBlog")
	hDefineTemplate(kCreateLink, 	"base", "narrow", "minFrame", "nuField", "createLink")
	hDefineTemplate(kCreatePoll, 	"base", "narrow", "minFrame", "nuField", "createPoll")

	//hDefineTemplate(kRegisterDetailsNopopup, "base", "narrow", "frame", "registerDetailsNopopup")

	// Popup forms (they do not inherit from 'base')
	hDefineTemplate(kNuFormPopup, 	"popupBase", "nuField", "nuForm", "defaultForm")
	hDefineTemplate(kLoginPopup, 	"popupBase", "nuField", "login")
	//hDefineTemplate(kLogin,			"base", "narrow", "frame",  "nuField", "login")

	hDefineTemplate(kLogin,			  "base", "narrow", "minFrame", "nuField", "login")
	hDefineTemplate(kRegister,		  "base", "narrow", "minFrame", "nuField", "register")
	hDefineTemplate(kRegisterDetails, "base", "narrow", "minFrame", "nuField", "registerDetails")


	hDefineTemplate(kRegisterPopup, 	"register")
	hDefineTemplate(kViewPollResults,	"viewPollResults", "comments")
	hDefineTemplate(kTestPopup, 		"testPopup")

	// Javascript snippets
	//tDefineTemplate(kRegisterDetailsScript, "registerDetailsScript")  // TODO: find a new home for this.  Just add to registerDetails(?)
}

///////////////////////////////////////////////////////////////////////////////
//
// program entry
//
///////////////////////////////////////////////////////////////////////////////
func init() {
	pr("init")

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
	http.HandleFunc("/testPopup/"	,   		hwrap(testPopupHandler))
	http.HandleFunc("/viewPollResults/",   		hwrap(viewPollResultsHandler))

	// Server static file.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Special handling for favicon.ico.
	http.Handle("/favicon.ico", http.FileServer(http.Dir("./static")))

	pr("Listening on http://localhost:" + flags.port + "...")
	http.ListenAndServe(":" + flags.port, nil)
}

func main() {
	pr("main")

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


