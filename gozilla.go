// gozilla - Golang implementation of votezilla

package main

import (
    "html/template"
    "log"
    "net/http"
)

type Page struct {
    Title string
    Body  []byte
}

var (
    templates *template.Template = nil
        debug = true
)

///////////////////////////////////////////////////////////////////////////////
//
// render template files
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
    var err error
    templates, err = template.ParseFiles("templates/frontPage.html",
                                         "templates/forgotPassword.html",
                                         "templates/login.html",
                                         "templates/register.html")
    if err != nil {
        log.Fatal(err)
    }
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
    log.Printf("renderTemplate: " + tmpl + ".html")
    
    if debug {
        parseTemplateFiles()
    }

    err := templates.ExecuteTemplate(w, tmpl + ".html", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}


///////////////////////////////////////////////////////////////////////////////
//
// frontPage
//
///////////////////////////////////////////////////////////////////////////////
func frontPageHandler(w http.ResponseWriter, r *http.Request) {
    var args struct{}
    renderTemplate(w, "frontPage", args)
}

///////////////////////////////////////////////////////////////////////////////
//
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {
    var args struct{}
    renderTemplate(w, "login", &args)
}

func postLoginHandler(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/", http.StatusFound)
}

///////////////////////////////////////////////////////////////////////////////
//
// forgotPassword
//
///////////////////////////////////////////////////////////////////////////////
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
    var args struct{}
    
    renderTemplate(w, "forgotPassword", args)
}

func postForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/", http.StatusFound)
}

///////////////////////////////////////////////////////////////////////////////
//
// register
//
///////////////////////////////////////////////////////////////////////////////
func registerHandler(w http.ResponseWriter, r *http.Request) {
    var args struct{}
    renderTemplate(w, "register", args)
}

func postRegisterHandler(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/", http.StatusFound)
}

///////////////////////////////////////////////////////////////////////////////
//
// program entry
//
///////////////////////////////////////////////////////////////////////////////
func init() {
    log.Printf("init")
    
    parseTemplateFiles()
}

func main() {
    log.Printf("main")
    
    http.HandleFunc("/",                frontPageHandler)

    http.HandleFunc("/login/",            loginHandler)
    http.HandleFunc("/forgotPassword/",    forgotPasswordHandler)
    http.HandleFunc("/register/",        registerHandler)
    
    http.HandleFunc("/postLogin/",            postLoginHandler)
    http.HandleFunc("/postForgotPassword/",    postForgotPasswordHandler)
    http.HandleFunc("/postRegister/",        postRegisterHandler)
    
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
        
    http.ListenAndServe(":8080", nil)
    
    log.Printf("Listening on http://localhost:8080...")
}    