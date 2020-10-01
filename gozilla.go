// gozilla.go
package main

import (
	"fmt"
    "net"
	"net/http"

	// Note: htemplate does HTML-escaping, which prevents against HTML-injection attacks!
	//       ttemplate does not, is not currently used and should not be used, but could be used for rendering HTML if absolutely necessary.
	//       To re-enable ttemplate, be sure to enable it everywhere, including in utils.go.
	htemplate "html/template"
	//ttemplate "text/template"
)

var (
	htemplates  map[string]*htemplate.Template
	//ttemplates  map[string]*ttemplate.Template

	err		 	error

	// NavMenu (constant)
	navMenu		[]string
)

const (
	kActivity = "activity"
	kArticle = "article"
	kCreate = "create"
	kCreateBlog = "createBlog"
	kCreateLink = "createLink"
	kCreatePoll = "createPoll"
	kLogin = "login"
	kLoginRequired = "loginRequired"	// Prompt to log in / sign in if required from user action.	
	kLoginSignup = "loginSignup"		// User clicks on log in / sign in button
	kNews = "news"
	kNewsSources = "newsSources"
	kNuForm = "nuForm"
	kNuFormPopup = "nuFormPopup"
	kRegister = "register"
	kRegisterDetails = "registerDetails"
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
	Metadata		map[string]string
}
// title - page title
// oImage - optional image ("" = use default votezilla image)
// oDescription - optional description ("" = use default description)
func makePageArgs(r *http.Request, title, oImage, oDescription string) (pa PageArgs) {
	pa.Title = title

	// Source: https://ogp.me/#types
	// Test: https://developers.facebook.com/tools/debug/?q=votezilla.io
	pa.Metadata = make(map[string]string)
	pa.Metadata["og:title"]			= title
	pa.Metadata["og:image"]		 	= ternary_str(oImage != "", oImage, "http://votezilla.io/static/votezilla logo/votezilla FB og image.jpg")
	pa.Metadata["og:description"] 	= ternary_str(oDescription != "", oDescription, `Votezilla:
a censorship-free social network based on creating polls, voting, sharing news, and fostering positive
political discussion. (Or nerd out on other topics you love.)`)
	pa.Metadata["og:type"] 		 	= "website"
	pa.Metadata["og:site_name"] 	= "Votezilla"
	pa.Metadata["og:image:type"] 	= "image/jpeg"
	pa.Metadata["og:locale"] 		= "en_US"
	pa.Metadata["og:url"]			= "http://votezilla.io" + r.URL.Path + "?" + r.URL.RawQuery
	pa.Metadata["fb:app_id"]		= "759729064806025"

	prVal("r.URL", r.URL)

	return pa
}

// Form Frame Args
type FormFrameArgs struct {
	PageArgs
	Form			Form
}
func makeFormFrameArgs(r *http.Request, form *Form, title string) FormFrameArgs {
	return FormFrameArgs {
		PageArgs: 		makePageArgs(r, title, "", ""),
		Form: 			*form,
	}
}

// Frame Args
type FrameArgs struct {
	PageArgs
	NavMenu				[]string
	UrlPath				string
	UserId				int64
	Username			string
	UpVotes				[]int64
	DownVotes			[]int64
}
func makeFrameArgs(r *http.Request, title, script, urlPath string, userId int64, username string) FrameArgs {
	pa := makePageArgs(r, title, "", "")
	pa.Script = script
	return FrameArgs {
		PageArgs: 		pa,
		NavMenu:		navMenu,
		UrlPath:		urlPath,
		UserId:			userId,
		Username:		username,
	}
}
func makeFrameArgs2(r *http.Request, title, script, urlPath string, userId int64, username string, upVotes, downVotes []int64) FrameArgs {
	pa := makeFrameArgs(r, title, script, urlPath, userId, username)
	pa.UpVotes   = upVotes
	pa.DownVotes = downVotes
	return pa
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


func widthHandler(w http.ResponseWriter, r *http.Request) {
	serveHTML(w, `
		<script>
			alert('Your device inner width: ' + window.innerWidth +
				  '; screen width: ' + screen.width);
		</script>
	`)
}

///////////////////////////////////////////////////////////////////////////////
//
// handler wrapper - Each request should refresh the session.
//
///////////////////////////////////////////////////////////////////////////////
func hwrap(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		prf("\n Handling request from: %s\n", formatRequest(r))
		startTimer("hwrap")

		err := CheckAndLogIP(r)
		if err != nil {
			serveError(w, err)
			return
		}

		handler(w, r)  // Handle the request.

		endTimer("hwrap")
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
		assertMsg(!found, fmt.Sprintf("Conflicting hDefineTemplate definition for %s!", handle))

		htemplates[handle] = htemplate.Must(htemplate.ParseFiles(map_str(getTemplatePath, filenames)...))
	}

	hDefineTemplate(kNews, 			"base", "wide", "frame", "news")

	hDefineTemplate(kNuForm, 		"base", "narrow", "frame", "nuField", "nuForm", "defaultForm")
	hDefineTemplate(kArticle, 		"base", "wide", "frame", "sidebar", "article", "comments")
	hDefineTemplate(kNewsSources,	"base", "wide", "frame", "newsSources")  // nyi
	hDefineTemplate(kActivity, 		"base", "wide", "frame", "activity")

	hDefineTemplate(kCreate, 		"base", "narrow", "minFrame", "nuField", "create")
	//hDefineTemplate(kCreate, 		"base", "wide", "frame", "nuField", "create")
	hDefineTemplate(kCreateBlog, 	"base", "narrow", "minFrame", "nuField", "createBlog")
	hDefineTemplate(kCreateLink, 	"base", "narrow", "minFrame", "nuField", "createLink")
	hDefineTemplate(kCreatePoll, 	"base", "narrowWithSidebar", "minFrame", "nuField", "createPoll")

	hDefineTemplate(kLogin,			  "base", "narrow", "minFrame", "nuField", "login")				// Log in
	hDefineTemplate(kLoginSignup,  	  "base", "narrow", "minFrame", "nuField", "loginSignup")		// Option to Log in or Sign up
	hDefineTemplate(kRegister,		  "base", "narrow", "minFrame", "nuField", "register")			// Sign up
	hDefineTemplate(kRegisterDetails, "base", "narrow", "minFrame", "nuField", "registerDetails")	// Sign up II: Demographics

	hDefineTemplate(kViewPollResults,	"base", "wide", "frame", "sidebar", "viewPollResults", "comments")

	// Pop-ups:
	hDefineTemplate(kTutorial, 			"tutorial")
	hDefineTemplate(kLoginRequired, "loginRequired")

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

///
type fileServer_Cached struct {
	fileServer	http.Handler
}
func FileServer_Cached(root http.FileSystem) http.Handler {
	return &fileServer_Cached{
		fileServer: http.FileServer(root),
	}
}
func (f *fileServer_Cached) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age:31536000, public")
	f.fileServer.ServeHTTP(w, r)
}

func SetupWebHandlers() *http.ServeMux {
	mux := &http.ServeMux{}

	mux.HandleFunc("/",                			hwrap(newsHandler))
	mux.HandleFunc("/ajaxCreateComment/",		hwrap(ajaxCreateComment))
	mux.HandleFunc("/ajaxCheckForNotifications/",hwrap(ajaxCheckForNotifications))
	mux.HandleFunc("/ajaxExpandComment/",		hwrap(ajaxExpandComment))
	mux.HandleFunc("/ajaxPollVote/",			hwrap(ajaxPollVote))
	mux.HandleFunc("/ajaxScrapeTitle/",			hwrap(ajaxScrapeTitle))
	mux.HandleFunc("/ajaxScrapeImageURLs/",		hwrap(ajaxScrapeImageURLs))
	mux.HandleFunc("/ajaxVote/",				hwrap(ajaxVote))
	mux.HandleFunc("/article/",       			hwrap(articleHandler))
	mux.HandleFunc("/activity/",       			hwrap(activityHandler))
	mux.HandleFunc("/create/",   				hwrap(createHandler))
	mux.HandleFunc("/createBlog/",   			hwrap(createBlogHandler))
	mux.HandleFunc("/createLink/",   			hwrap(createLinkHandler))
	mux.HandleFunc("/createPoll/",   			hwrap(createPollHandler))
	mux.HandleFunc("/history/",        			hwrap(historyHandler))
	mux.HandleFunc("/ip/",             			hwrap(ipHandler))
	mux.HandleFunc("/login/",          			hwrap(loginHandler))
	mux.HandleFunc("/loginRequired/",			hwrap(loginRequiredHandler))
	mux.HandleFunc("/loginSignup/",          	hwrap(loginSignupHandler))
	mux.HandleFunc("/logout/",         			hwrap(logoutHandler))
	mux.HandleFunc("/polls/",           		hwrap(pollsHandler))
	mux.HandleFunc("/news/",           			hwrap(newsHandler))
	mux.HandleFunc("/register/",       			hwrap(registerHandler))
	mux.HandleFunc("/registerDetails/",			hwrap(registerDetailsHandler))
	mux.HandleFunc("/tutorial/"	,   			hwrap(tutorialHandler))
	mux.HandleFunc("/updatePassword/", 			hwrap(updatePasswordHandler))
	mux.HandleFunc("/viewPollResults/",			hwrap(viewPollResultsHandler))
	mux.HandleFunc("/width/",					hwrap(widthHandler))

	// Serve static files.
	mux.Handle("/static/", http.StripPrefix("/static/", FileServer_Cached(http.Dir("./static"))))

	// Special handling for favicon.ico.
	mux.Handle("/favicon.ico", FileServer_Cached(http.Dir("./static")))

	return mux
}

func WebServer() {
	if flags.separateNewsAndPolls {
		navMenu		= []string{"polls", "news", "create", "activity", "history" }
	} else {
		navMenu		= []string{"news", "create", "activity", "about", "history" }
	}
	InitSecurity()
	InitNewsSources()
	InitFirewall()
	InitWebServer()
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
	} else if flags.cachingService != "" {
		CachingService()
	} else {
		WebServer()
	}
}


