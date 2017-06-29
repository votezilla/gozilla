// gozilla - Golang implementation of votezilla

package main

import (
    "bytes"
    "crypto/sha256"
    "database/sql"
    "encoding/binary"
    "flag"    
    "github.com/bluele/gforms"
    "github.com/lib/pq"
    "fmt"
    "html/template"
    "io"
    "log"
    "net/http"  
    "reflect"
)

var (
    db          *sql.DB
    templates   *template.Template
    err         error
    
    debug       string
    
    dbSalt = "SALT" // Database salt, for storing passwords safe from database leaks.
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

func printVal(label string, v interface{}) {
	log.Printf("%s: %v", label, v)
}

func printValX(label string, v interface{}) {
	log.Printf("%s: %x", label, v)
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
    
    if debug != "" {
        parseTemplateFiles()
    }

    err := templates.ExecuteTemplate(w, templateName + ".html", data)
    check(err)
}

func executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
    log.Printf("executeTemplate: " + templateName + ".html")
    
    if debug != "" {
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
    
    form := RegisterForm(r)
    tableForm := TableForm{
        form,
        "Register",
        "",
    }
    
    if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
        // Non-matching passwords
        if !MatchingPasswords(form) {
            tableForm.AdditionalError = "Passwords must match"
        } else { 
            // Passwords match, everything is good - Register the user
            
            // Parse POST data into "data".
            data := RegisterData{}
            form.MapTo(&data)
            
            // Use a hashed password for security.
            printVal("data.Password + dbSalt", data.Password + dbSalt)
            printVal("[]byte(data.Password + dbSalt)", []byte(data.Password + dbSalt))
            
            passwordHash := sha256.Sum256([]byte(data.Password + dbSalt))
            printVal("passwordHash:", passwordHash) 
            
            var passwordHashInts[4]int64
            err:= binary.Read(bytes.NewBuffer(passwordHash[:]), binary.LittleEndian, &passwordHashInts)
            check(err)
            printVal("passwordHashInts", passwordHashInts)
            
   			// SHIT, GOTTA SEND VERIFICATION EMAIL... USER DOESN'T GET CREATED UNTIL EMAIL GETS VERIFIED
   
            // INSERT IT FOR NOW, TODO: VERIFY EMAIL AND SET emailverified=True when email is verified
            
            // Works: INSERT INTO votezilla.user(username,passwordhash) VALUES('asmith', '{798798,-8980,2323,6546}');
            printVal("db", db)
            
            var lastInsertId int
            err = db.QueryRow(
                "INSERT INTO votezilla.user(username,passwordhash) VALUES ($1, $2) returning id;", 
                data.Username, 
                pq.Array(passwordHashInts),
            ).Scan(&lastInsertId)
            check(err)
            fmt.Println("lastInsertId =", lastInsertId)
            
            fmt.Fprintf(w, "form: %+v", form)
            fmt.Fprintf(w, "data: %+v", data)
            
            
            // Set logged-in cookie
            
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
    
       
    // Grab command line flags
    f1 := flag.String("dbname",     "votezilla", "Database to connect to")      ; 
    f2 := flag.String("dbuser",     "",          "Database user")               ; 
    f3 := flag.String("dbpassword", "",          "Database password")           ; 
    f4 := flag.String("dbsalt",     "",          "Database salt (for security)"); 
    f5 := flag.String("debug",      "",          "debug=true for development")  ; 
    flag.Parse()
    
    dbName      := *f1
    dbUser      := *f2
    dbPassword  := *f3
    dbSalt       = *f4
    debug        = *f5

    fmt.Println("dbName", dbName)
    fmt.Println("dbUser", dbUser)
    fmt.Println("dbPassword", dbPassword)
    fmt.Println("dbSalt", dbSalt)
    fmt.Println("debug", debug)

    // Connect to database
    dbInfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
        dbUser, dbPassword, dbName)  

    fmt.Printf("dbInfo: %s", dbInfo)

    db, err = sql.Open("postgres", dbInfo)
    fmt.Println("err:", err)
    check(err)
    
    printVal("db", db)
    
    if db != nil {
        defer db.Close()
    }

    http.HandleFunc("/",                frontPageHandler)

    http.HandleFunc("/login/",  loginHandler)
    http.HandleFunc("/forgotPassword/", forgotPasswordHandler)
    http.HandleFunc("/register/",   registerHandler)
    
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
        
    http.ListenAndServe(":8080", nil)
    
    log.Printf("Listening on http://localhost:8080...")
}   