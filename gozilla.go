// gozilla.go
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"net/http"
	"strings"
	"text/template" // Faster than "html/template", and less of a pain for safeHTML
)

var (
	templates   map[string]*template.Template
	err		 	error
	
	// NavMenu (constant)
	navMenu		= []string{"news", "submit", "history"}
	
	anonymityLevels = [][2]string { 
		{"R",	"Real name - Aaron Smith"},
		{"A",	"Alias - magicsquare666"},
		{"F",	"Random Anonymous Name - Wacky Panda"},
	}
)

const (
	//PollFlags
	pf_AnyoneCanAddOptions		= 1 << 0
	pf_CanSelectMultipleOptions = 1 << 1
)

// Template arguments for webpage template.
type PageArgs struct {
	Title			string
	Script			string
}

///////////////////////////////////////////////////////////////////////////////
//
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {	
	userId := GetSession(r)
	assert(userId == -1) // User must not be already logged in!
	
	form := LoginForm(r)
	
	if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
		// Parse POST data.
		data := LoginData{}
		form.MapTo(&data)
		
		var rows *sql.Rows
		if strings.Contains(data.EmailOrUsername, "@") {
			rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " + 
							"FROM $$User WHERE Email = $1;", 
							data.EmailOrUsername)
		} else {
			rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " + 
							"FROM $$User WHERE Username = $1;", 
							data.EmailOrUsername)
		}
		
		var userId int64
		var passwordHashInts int256			

		defer rows.Close()
		if !rows.Next() {
			field, err := form.GetField("email or username")
			assert(err)
			field.SetErrors([]string{"That email does not exist. Do you need to register?"})
		} else {
			err := rows.Scan(&userId, &passwordHashInts[0], &passwordHashInts[1], &passwordHashInts[2], &passwordHashInts[3]);
			check(err)
			check(rows.Err())
			prf(go_, "User found! - id: '%d' passwordHashInts: %#v\n", userId, passwordHashInts)	
		
			passwordHash := GetPasswordHash256(data.Password)		
			if  passwordHash[0] != passwordHashInts[0] ||
				passwordHash[1] != passwordHashInts[1] ||
				passwordHash[2] != passwordHashInts[2] ||
				passwordHash[3] != passwordHashInts[3] {

				field, err := form.GetField("password")
				assert(err)
				field.SetErrors([]string{"Invalid password.  Forgot password?"})	
			} else {				
				CreateSession(w, r, userId, data.RememberMe)

				http.Redirect(w, r, "/news?alert=LoggedIn", http.StatusSeeOther)   
				return	
			}
		}
	}
	
	// handle GET, or invalid form data from POST...	
	{
		args := FormArgs {
			PageArgs: PageArgs{Title: "Login"},
			Footer: `<a href="/forgotPassword">Forgot your password?</a>`,
			Forms: []TableForm{{
				Form: form,
				CallToAction: "Login",
			}}}
		executeTemplate(w, "form", args)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// logout
//
///////////////////////////////////////////////////////////////////////////////
func logoutHandler(w http.ResponseWriter, r *http.Request) {	
	DestroySession(w, r)
	
	http.Redirect(w, r, "/news?alert=LoggedOut", http.StatusSeeOther)   
	return
}

///////////////////////////////////////////////////////////////////////////////
//
// forgotPassword
//
///////////////////////////////////////////////////////////////////////////////
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement forgotPassword
}

///////////////////////////////////////////////////////////////////////////////
//
// register
//
///////////////////////////////////////////////////////////////////////////////
func registerHandler(w http.ResponseWriter, r *http.Request) {
	form := RegisterForm(r)
	tableForm := TableForm{
		Form: form,
		CallToAction: "Register",
	}

	if r.Method == "POST" && form.IsValid() { // Handle POST, with valid data...
		// Parse POST data.
		data := RegisterData{}
		form.MapTo(&data)

		// Use a hashed password for security.
		passwordHashInts := GetPasswordHash256(data.Password)

		// TODO: Gotta send verification email... user doesn't get created until email gets verified.
		// TODO: Verify email and set emailverified=True when email is verified
		
		// Check for duplicate email
		if DbExists("SELECT * FROM $$User WHERE Email = $1;", data.Email) {
			pr(go_, "That email is taken... have you registered already?")
			
			field, err := form.GetField("email")
			assert(err)
			field.SetErrors([]string{"That email is taken... have you registered already?"})
        } else { 
        	// Check for duplicate username
			if DbExists("SELECT * FROM $$User WHERE Username = $1;", data.Username) {
				pr(go_, "That username is taken... try another one.  Or, have you registered already?")
				field, err := form.GetField("username")
				assert(err)
				field.SetErrors([]string{"That username is taken... try another one.  Or, have you registered already?"})
			} else {
				// Add new user to the database   
				prf(go_, "passwordHashInts[0]: %T %#v\n", passwordHashInts[0], passwordHashInts[0])
				userId := DbInsert(
					"INSERT INTO $$User(Email, Username, PasswordHash) " +
					"VALUES ($1, $2, ARRAY[$3::bigint, $4::bigint, $5::bigint, $6::bigint]) returning id;", 
					data.Email,
					data.Username,
					passwordHashInts[0],
					passwordHashInts[1],
					passwordHashInts[2],
					passwordHashInts[3])
				
				// Create session (encrypted userId).
				CreateSession(w, r, userId, data.RememberMe)
				// Set "RememberMe" cookie
				if data.RememberMe {
					setCookie(w, r, "RememberMe", "true", longExpiration(), false)
				} else {
					setCookie(w, r, "RememberMe", "false", longExpiration(), false)
				}
				
				http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)
				return	
			}
		}
	}  

	// handle GET, or invalid form data from POST...	
	{
		args := FormArgs {
			PageArgs: PageArgs{Title: "Register"},
			Forms: []TableForm{
				tableForm,
		}}
		executeTemplate(w, "form", args)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// register details about the user
//
///////////////////////////////////////////////////////////////////////////////
func registerDetailsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)
	
	form := RegisterDetailsForm(r)
	
	userId := GetSession(r)			
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr(go_, "secure cookie not found")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	
	if r.Method == "POST" && form.IsValid() { // Handle POST, with valid data...
		// Passwords match, everything is good - Register the user

		// Parse POST data into "data".
		data := RegisterDetailsData{}
		form.MapTo(&data)
		
		prVal(go_, "userId", userId)
	
		// Update the user record with registration details.
		DbQuery(
			`UPDATE $$User
				SET (Name, Country, Location, BirthYear, Gender, Party, Race, Marital, Schooling)
				= ($2, $3, $4, $5, $6, $7, $8, $9, $10)
				WHERE Id = $1::bigint`, 
			userId,
			data.Name,
			data.Country,
			data.Location,
			data.BirthYear,
			data.Gender,
			data.Party,
			pq.Array(data.Races),
			data.Marital,
			data.Schooling)
		
		http.Redirect(w, r, "/news?alert=AccountCreated", http.StatusSeeOther)   
		return	
	} 
	
	// handle GET, or invalid form data from POST...	
	{
		// render registerDetailsScript template
		var scriptString string
		{
			scriptData := struct {
				CountriesWithStates			map[string]bool
				CountriesWithPostalCodes	map[string]bool
			}{
			    CountriesWithStates,
			    CountriesWithPostalCodes,
			}
			
			var scriptHTML bytes.Buffer
			renderTemplate(&scriptHTML, "registerDetailsScript", scriptData)
			scriptString = scriptHTML.String()
		}
		
		congrats := ""
		if r.Method == "GET" {
			congrats = "Congrats for registering" // Congrats for registering... now enter more information.
		}
		
		args := FormArgs {
			PageArgs: PageArgs{
				Title: "Voter Information",
				Script: scriptString},
			Congrats: congrats,
			Introduction: "A good voting system ensures everyone is represented.<br>" +
			              "Your information is confidential.",
			Forms: []TableForm{{
				Form: form,
				CallToAction: "Submit",
		}}}
		executeTemplate(w, "form", args)
	}
}

func registerDoneHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)
	serveHTML(w, `<h2>Congrats, you just registered</h2>
			      <script>alert('Congrats, you just registered')</script>`)
}

///////////////////////////////////////////////////////////////////////////////
//
// submit new post
//
///////////////////////////////////////////////////////////////////////////////
func submitHandler(w http.ResponseWriter, r *http.Request) {
	executeTemplate(w, "submit", PageArgs{Title: "Submit"})
}

func submitLinkHandler(w http.ResponseWriter, r *http.Request) {
	form := SubmitLinkForm(r)
	tableForm := TableForm{
		Form: form,
		CallToAction: "Submit",
	}
	
	userId := GetSession(r)			
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr(go_, "Must be logged in submit a post.  TODO: add submitLinkHandler to stack somehow.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	if r.Method == "POST" && form.IsValid() { // Handle POST, with valid data...

		// Parse POST data
		data := SubmitLinkData{}
		form.MapTo(&data)
		
		prVal(go_, "data", data)
		prVal(go_, "form", form)

		pr(go_, "Inserting new LinkPost into database.")
		//prf(go_, `INSERT INTO $$LinkPost(UserId, Title, LinkURL, Category) 
		//	      VALUES(%v::bigint, %v, %v) returning id;`, userId, data.Title, data.Link, data.Category)

		// Update the user record with registration details.
		newPostId := DbInsert(
			`INSERT INTO $$LinkPost(UserId, LinkURL, Title, Category, UrlToImage) 
			 VALUES($1::bigint, $2, $3, $4, $5) returning id;`, 
			userId,
			data.Link,
			data.Title,
			data.Category,
			data.Thumbnail)

		http.Redirect(w, r, fmt.Sprintf("/news?alert=SubmittedLink&newPostId=%d", newPostId), http.StatusSeeOther)   
		return	
	}  

	// handle GET, or invalid form data from POST...	
	{		
		/*type SubmitLinkArgs struct {
			FormArgs
			Categories	[]string
		}*/
		args := FormArgs{
			PageArgs:	PageArgs{Title: "Submit Link"},
			Forms:		[]TableForm{tableForm},
		}
		executeTemplate(w, "submitLink", args)
	}	
}

func submitPollHandler(w http.ResponseWriter, r *http.Request) {
	pr(go_, "submitPollHandler")

	userId := GetSession(r)			
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr(go_, "Must be logged in submit a post.  TODO: add submitPollHandler to stack somehow.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	prVal(go_, "r.Method", r.Method)
	
	form := makeForm(
		makeTextField("title", "Title:", "Ask something...", 50, 12, 255),
		makeTextField("option1", "Poll option 1:", "add option...", 50, 1, 255),
		makeTextField("option2", "Poll option 2:", "add option...", 50, 1, 255),
		makeBoolField("bAnyoneCanAddOptions", "Poll options:", "Allow anyone to add options", true),
		makeBoolField("bCanSelectMultipleOptions", "", "Allow people to select multiple options", true),
		makeSelectField("category", "Poll category:", newsCategoryInfo.CategorySelect, true, true),
		makeSelectField("anonymity", "Post As:", anonymityLevels, false, true),
	)
	
	// Add fields for additional options that were added, there could be an arbitrary number, we'll cap it at 1024 for now.
	pr(go_, "Adding additional poll options")
	pollOptions := []*Field{form["option1"], form["option2"]}
	
	// Just use brute force for now.  Don't break at the end, as we don't want the bricks to fall when someone erases the name of an option in the middle.  
	// TODO: optimize this later, if necessary, possibly with a hidden length field, if necessary.
	
	for i := 3; i < 1024; i++ {
		optionName := fmt.Sprintf("option%d", i)
		// TODO: How should this case work?  Could be used as a case for removing options, if poll is not yet live.
		//       Once live, options with votes should not be removable.
		//       Leave the ""'s in the list so the position within the array can map directly to votes and indexes.
		if r.FormValue(optionName) != "" {  
			prVal(go_, "Adding new poll option", optionName)
			form[optionName] = makeTextField(optionName, fmt.Sprintf("Poll option %d:", i), "add option...", 50, 1, 255)
			pollOptions = append(pollOptions, form[optionName])
		} 
	}

	prVal(go_, "r.Method", r.Method)
	prVal(go_, "r.PostForm", r.PostForm)
	prVal(go_, "form", form)
	
	if r.Method == "POST" && form.validateData(r) {
		prVal(go_, "Valid form!!", form)
		
		pr(go_, "Inserting new PollPost into database.")

		// Serialize all of the poll options and flags into variables that can be inserted into database.
		var pollOptionData PollOptionData
		for i := 1; i < 1024; i++ {
			value := r.FormValue(fmt.Sprintf("option%d", i)) 
			if value != "" {
				pollOptionData.Options = append(pollOptionData.Options, value)
			}
		}		
		pollOptionData.AnyoneCanAddOptions      = r.FormValue("bAnyoneCanAddOptions") != ""
		pollOptionData.CanSelectMultipleOptions = r.FormValue("bCanSelectMultipleOptions") != ""
		
		pollOptionsJson, err := json.Marshal(pollOptionData)
		check(err)
		
		prVal(go_, "pollOptionsJson", pollOptionsJson)
		
		// Create the new poll.
		pollPostId := DbInsert(
			`INSERT INTO $$PollPost(UserId, Title, Category, Language, Country, UrlToImage, 
			                        PollOptionData) 
			 VALUES($1::bigint, $2, $3, $4, $5, $6, 
			        $7) returning id;`, 
			userId,
			form["title"].Value,
			form["category"].Value,
			"en",
			"us",
			"http://localhost:8080/static/ballotbox.png", // TODO: generate poll url from image search
			pollOptionsJson,
		)
		prVal(go_, "Just added a poll #", pollPostId)

		http.Redirect(w, r, fmt.Sprintf("/news?alert=SubmittedPoll&pollPostId=%d", pollPostId), http.StatusSeeOther)   
		return
	} else if r.Method == "POST" {
		prVal(go_, "Invalid form!!", form)
	}

	// handle GET, or invalid form data from POST...	
	{
		type PollArgs struct {
			PageArgs
			Form
			PollOptions			[]*Field
			//Categories		[]string
			//AnonymityLevels	map[string]string
		}
		args := PollArgs{
			PageArgs:			PageArgs{Title: "Submit Poll"},
			Form:				form,
			PollOptions:		pollOptions,
			//Categories:			newsCategoryInfo.CategoryOrder,
			//AnonymityLevels:	anonymityLevels,
		}
		prVal(go_, "args", args)
		executeTemplate(w, "submitPoll", args)
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
	return func(w http.ResponseWriter, r *http.Request) {
		prf(go_, "\nHandling request from: %s\n", formatRequest(r))
    	
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

	templates = make(map[string]*template.Template)
	
	// HTML templates
	templates["form"]			= template.Must(template.ParseFiles(T("base"), T("form"), T("defaultForm")))
	templates["comments"]		= template.Must(template.ParseFiles(T("base"), T("frame"), T("comments")))
	templates["news"]			= template.Must(template.ParseFiles(T("base"), T("frame"), T("news")))
	templates["newsSources"]	= template.Must(template.ParseFiles(T("base"), T("newsSources")))
	templates["submit"]			= template.Must(template.ParseFiles(T("base"), T("submit")))
	templates["submitLink"]		= template.Must(template.ParseFiles(T("base"), T("form"), T("submitLink")))
	templates["submitPoll"]		= template.Must(template.ParseFiles(T("base"), T("submitPoll")))
	
	// Javascript snippets
	templates["registerDetailsScript"]	= template.Must(template.ParseFiles(T("registerDetailsScript")))
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
	http.HandleFunc("/news/",           		hwrap(newsHandler))
	http.HandleFunc("/history/",        		hwrap(historyHandler)) // <-- TODO: Implement this!
	http.HandleFunc("/comments/",       		hwrap(commentsHandler))
	http.HandleFunc("/forgotPassword/", 		hwrap(forgotPasswordHandler))
	http.HandleFunc("/ip/",             		hwrap(ipHandler))
	http.HandleFunc("/login/",          		hwrap(loginHandler))
	http.HandleFunc("/logout/",         		hwrap(logoutHandler))
//	http.HandleFunc("/newsSources/",    		hwrap(newsSourcesHandler))
	http.HandleFunc("/register/",       		hwrap(registerHandler))
	http.HandleFunc("/registerDetails/",		hwrap(registerDetailsHandler))
	http.HandleFunc("/registerDone/",   		hwrap(registerDoneHandler))
	http.HandleFunc("/submit/",   				hwrap(submitHandler))
	http.HandleFunc("/submitPoll/",   			hwrap(submitPollHandler))
	http.HandleFunc("/submitLink/",   			hwrap(submitLinkHandler))
	http.HandleFunc("/ajaxVote/",				hwrap(ajaxVoteHandler))
	http.HandleFunc("/ajaxScrapeImageURLs/",	hwrap(ajaxScrapeImageURLs))
	
	// Server static file.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	
	// Special handling for favicon.ico.
	http.Handle("/favicon.ico", http.FileServer(http.Dir("./static")))
	
	pr(go_, "Listening on http://localhost:" + flags.port + "...")
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


