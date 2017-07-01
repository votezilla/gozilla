package main

import (
	//"html/template"
	"text/template"
	"log"
	"net/http"
)

func check(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	var err error
		
	log.Println(1)
	T := func(page string) string {
		return "templates/" + page + ".html"
	}
	
	log.Println(4)
	tmpl := make(map[string]*template.Template)
	tmpl["test_index.html"] = template.Must(template.ParseFiles(T("test_base"), T("test_index")))
	//tmpl["test_index.html"] = template.Must(template.ParseFiles(T("base"), T("frontPage")))
	check(err)
	
	log.Println(5)
	var args struct{
		Title string//template.HTML
		TestMaliciousCode string
	}
	args.Title = `<h1>votezilla</h1>` //template.HTML(`<h1>votezilla</h1>`)
	args.TestMaliciousCode = template.HTMLEscapeString(`<h1>hi</h1><script>alert("Hello this is anonymous")</script>`)
	err = tmpl["test_index.html"].Execute(w, args)
	check(err)
}

func main() {
	log.Printf("main")

	http.HandleFunc("/", testHandler)
	
	http.ListenAndServe(":8080", nil)
	
	log.Printf("Listening on http://localhost:8080...")
}  