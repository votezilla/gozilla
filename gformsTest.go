package main

import (
    "fmt"
    "github.com/bluele/gforms"
    "log"
    "net/http"
    "text/template"
)

type User struct {
    Name   string  `gforms:"name"`
    Weight float32 `gforms:"weight"`
}

var (
    tplText = `<form method="post">{{range $i, $field := .Fields}}
  <label>{{$field.GetName}}: </label>{{$field.Html}}
  {{range $ei, $err := $field.Errors}}<label class="error">{{$err}}</label>{{end}}<br />
{{end}}<input type="submit">
</form>`
)

func main() {
    log.Printf("main")

    tpl := template.Must(template.New("tpl").Parse(tplText))

    userForm := gforms.DefineForm(gforms.NewFields(
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

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        form := userForm(r)
        if r.Method != "POST" {
            tpl.Execute(w, form)
            return
        }
        if !form.IsValid() {
            tpl.Execute(w, form)
            return
        }
        user := User{}
        form.MapTo(&user)
        fmt.Fprintf(w, "ok: %v", user)
    })

    http.ListenAndServe(":9000", nil)
    
    log.Printf("Listening on http://localhost:9000...")
}