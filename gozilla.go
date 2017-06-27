// gozilla - Golang implementation of votezilla

package main

import (
    "bytes"
    "github.com/bluele/gforms"
    "fmt"
    "html/template"
    "io"
    "log"
    "net/http"  
    "reflect"
)

var (
    templates *template.Template = nil
    
    debug = true
)

type TableForm struct {
    Form          *gforms.FormInstance
    SubmitText  string
    AdditionalError string
}

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
                                  "templates/register.html",
                                  "templates/tableForm.html")

    if err != nil {
        log.Fatal(err)
    }
}

func renderTemplate(w io.Writer, templateName string, data interface{}) {
    log.Printf("renderTemplate: " + templateName + ".html")
    
    if debug {
        parseTemplateFiles()
    }

    err := templates.ExecuteTemplate(w, templateName + ".html", data)
    check(err)
}

func executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
    log.Printf("executeTemplate: " + templateName + ".html")
    
    if debug {
        parseTemplateFiles()
    }

    err := templates.ExecuteTemplate(w, templateName + ".html", data)
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
    executeTemplate(w, "frontPage", args)
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
     
     userForm := gforms.DefineForm(gforms.NewFields(
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
     
     form := userForm(r)
     
     log.Printf("%v -> %s", form, reflect.TypeOf(form))
     
     tableForm := TableForm{
         form,
         "Register",
         "",
     }
     
     if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
         loginData := LoginData{}
         
         form.MapTo(&loginData)
         fmt.Fprintf(w, "loginData ok: %v", loginData)
         return   
     }  
     
     // handle GET, or invalid form data from POST...   
     {
         var formHTML bytes.Buffer
 
         renderTemplate(&formHTML, "tableForm", tableForm)
 
         args.FormHTML = formHTML.String()
 
         executeTemplate(w, "register", args)
    }
}

///////////////////////////////////////////////////////////////////////////////
//
// forgotPassword
//
///////////////////////////////////////////////////////////////////////////////
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
    var args struct{}
    
    executeTemplate(w, "forgotPassword", args)
}

///////////////////////////////////////////////////////////////////////////////
//
// register
//
///////////////////////////////////////////////////////////////////////////////
func registerHandler(w http.ResponseWriter, r *http.Request) {
    var args struct{
        FormHTML string
    }
    
    type LoginData struct {
        Username                string `gforms:"username"`
        Password                string `gforms:"password"`
        
        //Demographics
        Gender                  string `gforms:"are you"`
        Party                   string `gforms:"do you usually think of yourself as a"`
    }
    
    form := registerForm1(r)
    
    log.Printf("%v -> %s", form, reflect.TypeOf(form))
    
    tableForm := TableForm{
        form,
        "Register",
        "",
    }
    
    if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
        loginData := LoginData{}
        
        log.Printf("pw: %s confirm_pw: %s", 
            form.Data["password"].RawStr, 
            form.Data["confirm password"].RawStr)
        
        // Non-matching passwords
        if form.Data["password"].RawStr != form.Data["confirm password"].RawStr {
            tableForm.AdditionalError = "Passwords must match"
        } else { 
            // Passwords match, everything is good - register the user
            form.MapTo(&loginData)
            fmt.Fprintf(w, "form: %+v", form)
            fmt.Fprintf(w, "loginData: %+v", loginData)
            return    
        }
    }  
    
    // handle GET, or invalid form data from POST...    
    {
        var formHTML bytes.Buffer

        renderTemplate(&formHTML, "tableForm", tableForm)

        args.FormHTML = formHTML.String()

        executeTemplate(w, "register", args)
    }
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

    http.HandleFunc("/login/",  loginHandler)
    http.HandleFunc("/forgotPassword/", forgotPasswordHandler)
    http.HandleFunc("/register/",   registerHandler)
    
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
        
    http.ListenAndServe(":8080", nil)
    
    log.Printf("Listening on http://localhost:8080...")
}   