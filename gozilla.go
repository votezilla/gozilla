// gozilla - Golang implementation of votezilla

package main

import (
    "bytes"
    "github.com/bluele/gforms"
    "fmt"
    "html/template"
    "log"
    "net/http"    
)

var (
    userForm gforms.Form

    // form template for auto-generating HTML for a form, in table-based layout
    tplText = `
<form method="post">
  <table border="0">
    {{range $i, $field := .Fields}}
      <tr>
        <td>{{$field.GetName}}:</td>
        <td>
          {{$field.Html | safeHTML}}
          {{range $ei, $err := $field.Errors}}
            <label class="error">{{$err}}</label>
          {{end}}
        </td>
      </tr>
    {{end}}
  </table>
  <br>
  <input type="submit" value="create account">
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
    
    t := template.New("").Funcs(
        template.FuncMap { 
            "safeHTML": func(x string) interface{} { return template.HTML(x) }})

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
        FormHTML string
    }
    
    type LoginData struct {
        Username string `gforms:"username"`
        Password string `gforms:"password"`
    }
    
    userForm = gforms.DefineForm(gforms.NewFields(
        gforms.NewTextField(
            "username",
            gforms.Validators{
                gforms.Required(),
                gforms.MaxLengthValidator(32),
            },
            gforms.TextInputWidget(map[string]string{
                "autocorrect": "off",
                "spellcheck": "false",
                "autocapitalize": "off",
                "autofocus": "true",
            }),
        ),
        gforms.NewTextField(
            "password",
            gforms.Validators{
                gforms.Required(),
                gforms.MinLengthValidator(4),
                gforms.MaxLengthValidator(16),
            },
            gforms.PasswordInputWidget(map[string]string{}),
        ),
    ))

    tpl := template.New("tpl").Funcs(
            template.FuncMap { 
                "safeHTML": func(x string) interface{} { return template.HTML(x) }})
    tpl, _ = tpl.Parse(tplText)

    form := userForm(r)
    
    if r.Method != "POST" || !form.IsValid() { // handle GET, or invalid form data...    
        var formHTML bytes.Buffer
        
        check(tpl.Execute(&formHTML, form))
        
        args.FormHTML = formHTML.String()
        
        log.Printf("processed form buffer: %s\n", args.FormHTML)
        
        log.Printf("form: %v\n", form)

        renderTemplate(w, "login", args)
    } else { // handle POST, with valid data...
        loginData := LoginData{}
        form.MapTo(&loginData)
        fmt.Fprintf(w, "loginData ok: %v", loginData)
    }
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

///////////////////////////////////////////////////////////////////////////////
//
// register
//
///////////////////////////////////////////////////////////////////////////////
func registerHandler(w http.ResponseWriter, r *http.Request) {
    var args struct{}
    renderTemplate(w, "register", args)
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

    http.HandleFunc("/login/",          loginHandler)
    http.HandleFunc("/forgotPassword/", forgotPasswordHandler)
    http.HandleFunc("/register/",       registerHandler)
    
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
        
    http.ListenAndServe(":8080", nil)
    
    log.Printf("Listening on http://localhost:8080...")
}    