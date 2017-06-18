// gozilla - Golang implementation of votezilla

package main

import (
    "bytes"
    "github.com/bluele/gforms"
    "html/template"
    "log"
    "net/http"    
)

type LoginForm struct {
    Username string  `gforms:"username"`
    Password float32 `gforms:"password"`
}

var (
    userForm gforms.Form

    tplText = `
<form method="post">
  {{range $i, $field := .Fields}}
    <label>{{$field.GetName}}: </label>{{$field.Html | safeHTML}}
      {{range $ei, $err := $field.Errors}}
        <label class="error">{{$err}}</label>
      {{end}}
    <br />
  {{end}}<input type="submit">
</form>`
)

var (
    templates *template.Template = nil
    debug = true
)

///////////////////////////////////////////////////////////////////////////////
//
// utility functions
//
///////////////////////////////////////////////////////////////////////////////
func check(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

///////////////////////////////////////////////////////////////////////////////
//
// render template files
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
    var err error
    
    t := template.New("").Funcs(template.FuncMap { "safeHTML": func(x string) interface{} { log.Printf("safeHTML\n"); return template.HTML(x) } })

    templates, err = t.ParseFiles("templates/frontPage.html",
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
    var args struct{
        //formHTML bytes.Buffer
        FormHTML string
    }
    //renderTemplate(w, "login", &args)
    
    userForm = gforms.DefineForm(gforms.NewFields(
        gforms.NewTextField(
            "name",
            gforms.Validators{
                gforms.Required(),
                gforms.MaxLengthValidator(32),
            },
        ),
        gforms.NewFloatField(
            "weight",
            gforms.Validators{},
        ),
    ))

    //var err error
    tpl := template.New("tpl").Funcs(
            template.FuncMap { 
                "safeHTML": func(x string) interface{} { log.Printf("safeHTML\n"); return template.HTML(x) }},)
    tpl, _ = tpl.Parse(tplText)

    //tpl := template.Must(template.New("tpl").Parse(tplText))
    log.Printf("tpl: %v\n", tpl)
   // log.Printf("tpl: %s\n", string(tpl))
    
    form := userForm(r)
    
    if r.Method != "POST" {
        //tpl.Execute(w, form)
        //return
        
        var formHTML bytes.Buffer
        
        check(tpl.Execute(&formHTML, form))
        
        args.FormHTML = formHTML.String()
        
        log.Printf("processed form buffer: %s\n", args.FormHTML)
    }
    
    log.Printf("form: %v\n", form)
    
    //renderTemplate(w, "login", &form)
    renderTemplate(w, "login", args)
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