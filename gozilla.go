// gozilla.go
package main

import (
	"fmt"
	"net/http"

	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, is not currently used and should not be used, but could be used for rendering HTML if absolutely necessary.
	//       To re-enable ttemplate, be sure to enable it everywhere, including in utils.go.
	htemplate "html/template"
	//ttemplate "text/template"

    "net"
)

var (
	htemplates  map[string]*htemplate.Template
	//ttemplates  map[string]*ttemplate.Template

	err		 	error

	// NavMenu (constant)
	navMenu		= []string{"news", "create", "activity" }
)




const (
	kActivity = "activity"
	kArticle = "article"
	kCreate = "create"
	kCreateBlog = "createBlog"
	kCreateLink = "createLink"
	kCreatePoll = "createPoll"
	kLogin = "login"
	kNews = "news"
	kNewsSources = "newsSources"
	kNuForm = "nuForm"
	kNuFormPopup = "nuFormPopup"
	kRegister = "register"
	kRegisterDetails = "registerDetails"
	kTestPopup = "testPopup"
	kTutorial = "tutorial"
	kViewPollResults = "viewPollResults"
)

///////////////////////////////////////////////////////////////////////////////
//
// HTML Template Args
//
///////////////////////////////////////////////////////////////////////////////
// Page Args
type PageArgs struct {
	Title			string
	Script			string
}

// Form Frame Args
type FormFrameArgs struct {
	PageArgs
	Form			Form
}
func makeFormFrameArgs(form *Form, title string) FormFrameArgs {
	return FormFrameArgs {
		PageArgs: 		PageArgs{Title: title},
		Form: 			*form,
	}
}

// Frame Args
type FrameArgs struct {
	PageArgs
	NavMenu			[]string
	UrlPath			string
	UserId			int64
	Username		string
	UpVotes			[]int64
	DownVotes		[]int64
}
func makeFrameArgs(title, script, urlPath string, userId int64, username string) FrameArgs {
	return FrameArgs {
		PageArgs: 		PageArgs{Title: title, Script: script},
		NavMenu:		navMenu,
		UrlPath:		urlPath,
		UserId:			userId,
		Username:		username,
	}
}
func makeFrameArgs2(title, script, urlPath string, userId int64, username string, upVotes, downVotes []int64) FrameArgs {
	return FrameArgs {
		PageArgs: 		PageArgs{Title: title, Script: script},
		NavMenu:		navMenu,
		UrlPath:		urlPath,
		UserId:			userId,
		Username:		username,
		UpVotes:		upVotes,
		DownVotes:		downVotes,
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// TODO: get user's ip address
//       1) To log in the database when user is first created.
//		 2) To set their location in registerDetails and save them time.
// USING: https://play.golang.org/p/Z6ATIgo_IM
//        https://stackoverflow.com/questions/27234861/correct-way-of-getting-clients-ip-addresses-from-http-request-golang
//
///////////////////////////////////////////////////////////////////////////////
func ipHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "<p>remote addr: %s</p>", r.RemoteAddr)
	fmt.Fprintf(w, "<p>forwarded for: %s</p>", r.Header.Get("X-Forwarded-For"))
	fmt.Fprintf(w, "<br><p>r: %+v</p>", r)

	ip, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fmt.Fprintf(w, "userip: %q is not IP:port", r.RemoteAddr)
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		//return nil, fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)
		fmt.Fprintf(w, "userip: %q is not IP:port", r.RemoteAddr)
    }
    fmt.Fprintf(w, "<p>User IP: %s</p>", userIP)

    // This will only be defined when site is accessed via non-anonymous proxy
    // and takes precedence over RemoteAddr
    // Header.Get is case-insensitive
    forward := r.Header.Get("X-Forwarded-For")

    fmt.Fprintf(w, "<p>IP: %s</p>", ip)
    fmt.Fprintf(w, "<p>Port: %s</p>", port)
    fmt.Fprintf(w, "<p>Forwarded for: %s</p>", forward)
}

///////////////////////////////////////////////////////////////////////////////
//
// handler wrapper - Each request should refresh the session.
//
///////////////////////////////////////////////////////////////////////////////
func hwrap(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		prf("\n Handling request from: %s\n", formatRequest(r))

		err := CheckAndLogIP(r)
		if err != nil {
			serveError(w, err)
			return
		}

		handler(w, r)  // Handle the request.
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// parse template files - Parses the HTML template files.
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, but is necessary for rendering HTML, such as auto-generated forms.
	htemplates = make(map[string]*htemplate.Template)
	//ttemplates = make(map[string]*ttemplate.Template)

	getTemplatePath := func(page string) string {
		return "templates/" + page + ".html"
	}
	// We're trying to just use hDefineTemplate, since it prevents against HTML injection.
	//   Templates make it possible to use hDefineTemplate.
	//   Do it this way if at all possible!!!
	//
	//tDefineTemplate := func(handle string, filenames ...string) {
	//	ttemplates[handle] = ttemplate.Must(ttemplate.ParseFiles(map_str(getTemplatePath, filenames)...))
	//}
	hDefineTemplate := func(handle string, filenames ...string) {
		_, found := htemplates[handle]
		assertMsg(!found, "Conflicting hDefineTemplate definition!!!")

		htemplates[handle] = htemplate.Must(htemplate.ParseFiles(map_str(getTemplatePath, filenames)...))
	}

	hDefineTemplate(kNews, 			"base", "wide", "frame", "news")

	hDefineTemplate(kNuForm, 		"base", "narrow", "frame", "nuField", "nuForm", "defaultForm")
	hDefineTemplate(kArticle, 		"base", "wide", "frame", "sidebar", "article", "comments")
	hDefineTemplate(kNewsSources,	"base", "wide", "frame", "newsSources")  // nyi
	hDefineTemplate(kActivity, 		"base", "wide", "frame", "activity")

	hDefineTemplate(kCreate, 		"base", "narrow", "minFrame", "nuField", "create")
	hDefineTemplate(kCreateBlog, 	"base", "narrow", "minFrame", "nuField", "createBlog")
	hDefineTemplate(kCreateLink, 	"base", "narrow", "minFrame", "nuField", "createLink")
	hDefineTemplate(kCreatePoll, 	"base", "narrow", "minFrame", "nuField", "createPoll")

	hDefineTemplate(kLogin,			  "base", "narrow", "minFrame", "nuField", "login")
	hDefineTemplate(kRegister,		  "base", "narrow", "minFrame", "nuField", "register")
	hDefineTemplate(kRegisterDetails, "base", "narrow", "minFrame", "nuField", "registerDetails")

	hDefineTemplate(kViewPollResults,	"base", "wide", "frame", "sidebar", "viewPollResults", "comments")

	// Pop-ups:
	hDefineTemplate(kTestPopup, 		"testPopup")
	hDefineTemplate(kTutorial, 			"tutorial")

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

func SetupWebHandlers() *http.ServeMux {
	mux := &http.ServeMux{}

	mux.HandleFunc("/",                		hwrap(newsHandler))
	mux.HandleFunc("/ajaxCreateComment/",	hwrap(ajaxCreateComment))
	mux.HandleFunc("/ajaxExpandComment/",	hwrap(ajaxExpandComment))
	mux.HandleFunc("/ajaxPollVote/",		hwrap(ajaxPollVote))
	mux.HandleFunc("/ajaxScrapeTitle/",		hwrap(ajaxScrapeTitle))
	mux.HandleFunc("/ajaxScrapeImageURLs/",	hwrap(ajaxScrapeImageURLs))
	mux.HandleFunc("/ajaxVote/",			hwrap(ajaxVote))
	mux.HandleFunc("/article/",       		hwrap(articleHandler))
	mux.HandleFunc("/activity/",       		hwrap(activityHandler))
	mux.HandleFunc("/create/",   			hwrap(createHandler))
	mux.HandleFunc("/createBlog/",   		hwrap(createBlogHandler))
	mux.HandleFunc("/createLink/",   		hwrap(createLinkHandler))
	mux.HandleFunc("/createPoll/",   		hwrap(createPollHandler))
	mux.HandleFunc("/history/",        		hwrap(historyHandler))
	mux.HandleFunc("/ip/",             		hwrap(ipHandler))
	mux.HandleFunc("/login/",          		hwrap(loginHandler))
	mux.HandleFunc("/logout/",         		hwrap(logoutHandler))
	mux.HandleFunc("/news/",           		hwrap(newsHandler))
	mux.HandleFunc("/register/",       		hwrap(registerHandler))
	mux.HandleFunc("/registerDetails/",		hwrap(registerDetailsHandler))
	mux.HandleFunc("/testPopup/"	,   	hwrap(testPopupHandler))
	mux.HandleFunc("/tutorial/"	,   		hwrap(tutorialHandler))
	mux.HandleFunc("/updatePassword/", 		hwrap(updatePasswordHandler))
	mux.HandleFunc("/viewPollResults/",		hwrap(viewPollResultsHandler))

	// Serve static files.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Special handling for favicon.ico.
	mux.Handle("/favicon.ico", http.FileServer(http.Dir("./static")))

	return mux
}

func WebServer() {
	InitSecurity()
	InitNewsSources()
	InitFirewall()
	InitWebServer2()
}

func main() {
	pr("main")

	parseCommandLineFlags()

	OpenDatabase()
	defer CloseDatabase()

	if flags.imageService != "" {
		ImageService()
	} else if flags.newsService != "" {
		NewsService()
	} else {
		WebServer()
	}
}


