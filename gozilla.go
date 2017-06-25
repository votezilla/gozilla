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
	Form		  *gforms.FormInstance
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
		Username 				string `gforms:"username"`
		Password 				string `gforms:"password"`
		
		//Demographics
		Gender					string `gforms:"are you"`
		Party					string `gforms:"do you usually think of yourself as a"`
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
		gforms.NewTextField(
			"confirm password",
			gforms.Validators{},
			gforms.PasswordInputWidget(map[string]string{}),
		),
		gforms.NewTextField( //NewFloatField(
			"year of birth",
			gforms.Validators{
				gforms.Required(),
				gforms.MinLengthValidator(4),
				gforms.MaxLengthValidator(4),
		}),
		gforms.NewTextField(
			"zip code",
			gforms.Validators{
				gforms.MaxLengthValidator(5),
		}),		
		gforms.NewTextField(
			"country",
			gforms.Validators{
				gforms.Required(),
			},
			gforms.SelectWidget(
				map[string]string{},
				func() gforms.SelectOptions {
					return gforms.StringSelectOptions([][]string{
						// ISO-3166-1 Alpha-2 country list.  See: https://www.freeformatter.com/iso-country-list-html-select.html
						{"-", "-", "false", "false"},
						{"United States", "US", "false", "false"},
						{"United States Minor Outlying Islands", "UM", "false", "false"},
						{"Afghanistan", "AF", "false", "false"},
						{"Åland Islands", "AX", "false", "false"},
						{"Albania", "AL", "false", "false"},
						{"Algeria", "DZ", "false", "false"},
						{"American Samoa", "AS", "false", "false"},
						{"Andorra", "AD", "false", "false"},
						{"Angola", "AO", "false", "false"},
						{"Anguilla", "AI", "false", "false"},
						{"Antarctica", "AQ", "false", "false"},
						{"Antigua and Barbuda", "AG", "false", "false"},
						{"Argentina", "AR", "false", "false"},
						{"Armenia", "AM", "false", "false"},
						{"Aruba", "AW", "false", "false"},
						{"Australia", "AU", "false", "false"},
						{"Austria", "AT", "false", "false"},
						{"Azerbaijan", "AZ", "false", "false"},
						{"Bahamas", "BS", "false", "false"},
						{"Bahrain", "BH", "false", "false"},
						{"Bangladesh", "BD", "false", "false"},
						{"Barbados", "BB", "false", "false"},
						{"Belarus", "BY", "false", "false"},
						{"Belgium", "BE", "false", "false"},
						{"Belize", "BZ", "false", "false"},
						{"Benin", "BJ", "false", "false"},
						{"Bermuda", "BM", "false", "false"},
						{"Bhutan", "BT", "false", "false"},
						{"Bolivia, Plurinational State of", "BO", "false", "false"},
						{"Bonaire, Sint Eustatius and Saba", "BQ", "false", "false"},
						{"Bosnia and Herzegovina", "BA", "false", "false"},
						{"Botswana", "BW", "false", "false"},
						{"Bouvet Island", "BV", "false", "false"},
						{"Brazil", "BR", "false", "false"},
						{"British Indian Ocean Territory", "IO", "false", "false"},
						{"Brunei Darussalam", "BN", "false", "false"},
						{"Bulgaria", "BG", "false", "false"},
						{"Burkina Faso", "BF", "false", "false"},
						{"Burundi", "BI", "false", "false"},
						{"Cambodia", "KH", "false", "false"},
						{"Cameroon", "CM", "false", "false"},
						{"Canada", "CA", "false", "false"},
						{"Cape Verde", "CV", "false", "false"},
						{"Cayman Islands", "KY", "false", "false"},
						{"Central African Republic", "CF", "false", "false"},
						{"Chad", "TD", "false", "false"},
						{"Chile", "CL", "false", "false"},
						{"China", "CN", "false", "false"},
						{"Christmas Island", "CX", "false", "false"},
						{"Cocos (Keeling) Islands", "CC", "false", "false"},
						{"Colombia", "CO", "false", "false"},
						{"Comoros", "KM", "false", "false"},
						{"Congo", "CG", "false", "false"},
						{"Congo, the Democratic Republic of the", "CD", "false", "false"},
						{"Cook Islands", "CK", "false", "false"},
						{"Costa Rica", "CR", "false", "false"},
						{"Côte d'Ivoire", "CI", "false", "false"},
						{"Croatia", "HR", "false", "false"},
						{"Cuba", "CU", "false", "false"},
						{"Curaçao", "CW", "false", "false"},
						{"Cyprus", "CY", "false", "false"},
						{"Czech Republic", "CZ", "false", "false"},
						{"Denmark", "DK", "false", "false"},
						{"Djibouti", "DJ", "false", "false"},
						{"Dominica", "DM", "false", "false"},
						{"Dominican Republic", "DO", "false", "false"},
						{"Ecuador", "EC", "false", "false"},
						{"Egypt", "EG", "false", "false"},
						{"El Salvador", "SV", "false", "false"},
						{"Equatorial Guinea", "GQ", "false", "false"},
						{"Eritrea", "ER", "false", "false"},
						{"Estonia", "EE", "false", "false"},
						{"Ethiopia", "ET", "false", "false"},
						{"Falkland Islands (Malvinas)", "FK", "false", "false"},
						{"Faroe Islands", "FO", "false", "false"},
						{"Fiji", "FJ", "false", "false"},
						{"Finland", "FI", "false", "false"},
						{"France", "FR", "false", "false"},
						{"French Guiana", "GF", "false", "false"},
						{"French Polynesia", "PF", "false", "false"},
						{"French Southern Territories", "TF", "false", "false"},
						{"Gabon", "GA", "false", "false"},
						{"Gambia", "GM", "false", "false"},
						{"Georgia", "GE", "false", "false"},
						{"Germany", "DE", "false", "false"},
						{"Ghana", "GH", "false", "false"},
						{"Gibraltar", "GI", "false", "false"},
						{"Greece", "GR", "false", "false"},
						{"Greenland", "GL", "false", "false"},
						{"Grenada", "GD", "false", "false"},
						{"Guadeloupe", "GP", "false", "false"},
						{"Guam", "GU", "false", "false"},
						{"Guatemala", "GT", "false", "false"},
						{"Guernsey", "GG", "false", "false"},
						{"Guinea", "GN", "false", "false"},
						{"Guinea-Bissau", "GW", "false", "false"},
						{"Guyana", "GY", "false", "false"},
						{"Haiti", "HT", "false", "false"},
						{"Heard Island and McDonald Islands", "HM", "false", "false"},
						{"Holy See (Vatican City State)", "VA", "false", "false"},
						{"Honduras", "HN", "false", "false"},
						{"Hong Kong", "HK", "false", "false"},
						{"Hungary", "HU", "false", "false"},
						{"Iceland", "IS", "false", "false"},
						{"India", "IN", "false", "false"},
						{"Indonesia", "ID", "false", "false"},
						{"Iran, Islamic Republic of", "IR", "false", "false"},
						{"Iraq", "IQ", "false", "false"},
						{"Ireland", "IE", "false", "false"},
						{"Isle of Man", "IM", "false", "false"},
						{"Israel", "IL", "false", "false"},
						{"Italy", "IT", "false", "false"},
						{"Jamaica", "JM", "false", "false"},
						{"Japan", "JP", "false", "false"},
						{"Jersey", "JE", "false", "false"},
						{"Jordan", "JO", "false", "false"},
						{"Kazakhstan", "KZ", "false", "false"},
						{"Kenya", "KE", "false", "false"},
						{"Kiribati", "KI", "false", "false"},
						{"Korea, Democratic People's Republic of", "KP", "false", "false"},
						{"Korea, Republic of", "KR", "false", "false"},
						{"Kuwait", "KW", "false", "false"},
						{"Kyrgyzstan", "KG", "false", "false"},
						{"Lao People's Democratic Republic", "LA", "false", "false"},
						{"Latvia", "LV", "false", "false"},
						{"Lebanon", "LB", "false", "false"},
						{"Lesotho", "LS", "false", "false"},
						{"Liberia", "LR", "false", "false"},
						{"Libya", "LY", "false", "false"},
						{"Liechtenstein", "LI", "false", "false"},
						{"Lithuania", "LT", "false", "false"},
						{"Luxembourg", "LU", "false", "false"},
						{"Macao", "MO", "false", "false"},
						{"Macedonia, the former Yugoslav Republic of", "MK", "false", "false"},
						{"Madagascar", "MG", "false", "false"},
						{"Malawi", "MW", "false", "false"},
						{"Malaysia", "MY", "false", "false"},
						{"Maldives", "MV", "false", "false"},
						{"Mali", "ML", "false", "false"},
						{"Malta", "MT", "false", "false"},
						{"Marshall Islands", "MH", "false", "false"},
						{"Martinique", "MQ", "false", "false"},
						{"Mauritania", "MR", "false", "false"},
						{"Mauritius", "MU", "false", "false"},
						{"Mayotte", "YT", "false", "false"},
						{"Mexico", "MX", "false", "false"},
						{"Micronesia, Federated States of", "FM", "false", "false"},
						{"Moldova, Republic of", "MD", "false", "false"},
						{"Monaco", "MC", "false", "false"},
						{"Mongolia", "MN", "false", "false"},
						{"Montenegro", "ME", "false", "false"},
						{"Montserrat", "MS", "false", "false"},
						{"Morocco", "MA", "false", "false"},
						{"Mozambique", "MZ", "false", "false"},
						{"Myanmar", "MM", "false", "false"},
						{"Namibia", "NA", "false", "false"},
						{"Nauru", "NR", "false", "false"},
						{"Nepal", "NP", "false", "false"},
						{"Netherlands", "NL", "false", "false"},
						{"New Caledonia", "NC", "false", "false"},
						{"New Zealand", "NZ", "false", "false"},
						{"Nicaragua", "NI", "false", "false"},
						{"Niger", "NE", "false", "false"},
						{"Nigeria", "NG", "false", "false"},
						{"Niue", "NU", "false", "false"},
						{"Norfolk Island", "NF", "false", "false"},
						{"Northern Mariana Islands", "MP", "false", "false"},
						{"Norway", "NO", "false", "false"},
						{"Oman", "OM", "false", "false"},
						{"Pakistan", "PK", "false", "false"},
						{"Palau", "PW", "false", "false"},
						{"Palestinian Territory, Occupied", "PS", "false", "false"},
						{"Panama", "PA", "false", "false"},
						{"Papua New Guinea", "PG", "false", "false"},
						{"Paraguay", "PY", "false", "false"},
						{"Peru", "PE", "false", "false"},
						{"Philippines", "PH", "false", "false"},
						{"Pitcairn", "PN", "false", "false"},
						{"Poland", "PL", "false", "false"},
						{"Portugal", "PT", "false", "false"},
						{"Puerto Rico", "PR", "false", "false"},
						{"Qatar", "QA", "false", "false"},
						{"Réunion", "RE", "false", "false"},
						{"Romania", "RO", "false", "false"},
						{"Russian Federation", "RU", "false", "false"},
						{"Rwanda", "RW", "false", "false"},
						{"Saint Barthélemy", "BL", "false", "false"},
						{"Saint Helena, Ascension and Tristan da Cunha", "SH", "false", "false"},
						{"Saint Kitts and Nevis", "KN", "false", "false"},
						{"Saint Lucia", "LC", "false", "false"},
						{"Saint Martin (French part)", "MF", "false", "false"},
						{"Saint Pierre and Miquelon", "PM", "false", "false"},
						{"Saint Vincent and the Grenadines", "VC", "false", "false"},
						{"Samoa", "WS", "false", "false"},
						{"San Marino", "SM", "false", "false"},
						{"Sao Tome and Principe", "ST", "false", "false"},
						{"Saudi Arabia", "SA", "false", "false"},
						{"Senegal", "SN", "false", "false"},
						{"Serbia", "RS", "false", "false"},
						{"Seychelles", "SC", "false", "false"},
						{"Sierra Leone", "SL", "false", "false"},
						{"Singapore", "SG", "false", "false"},
						{"Sint Maarten (Dutch part)", "SX", "false", "false"},
						{"Slovakia", "SK", "false", "false"},
						{"Slovenia", "SI", "false", "false"},
						{"Solomon Islands", "SB", "false", "false"},
						{"Somalia", "SO", "false", "false"},
						{"South Africa", "ZA", "false", "false"},
						{"South Georgia and the South Sandwich Islands", "GS", "false", "false"},
						{"South Sudan", "SS", "false", "false"},
						{"Spain", "ES", "false", "false"},
						{"Sri Lanka", "LK", "false", "false"},
						{"Sudan", "SD", "false", "false"},
						{"Suriname", "SR", "false", "false"},
						{"Svalbard and Jan Mayen", "SJ", "false", "false"},
						{"Swaziland", "SZ", "false", "false"},
						{"Sweden", "SE", "false", "false"},
						{"Switzerland", "CH", "false", "false"},
						{"Syrian Arab Republic", "SY", "false", "false"},
						{"Taiwan, Province of China", "TW", "false", "false"},
						{"Tajikistan", "TJ", "false", "false"},
						{"Tanzania, United Republic of", "TZ", "false", "false"},
						{"Thailand", "TH", "false", "false"},
						{"Timor-Leste", "TL", "false", "false"},
						{"Togo", "TG", "false", "false"},
						{"Tokelau", "TK", "false", "false"},
						{"Tonga", "TO", "false", "false"},
						{"Trinidad and Tobago", "TT", "false", "false"},
						{"Tunisia", "TN", "false", "false"},
						{"Turkey", "TR", "false", "false"},
						{"Turkmenistan", "TM", "false", "false"},
						{"Turks and Caicos Islands", "TC", "false", "false"},
						{"Tuvalu", "TV", "false", "false"},
						{"Uganda", "UG", "false", "false"},
						{"Ukraine", "UA", "false", "false"},
						{"United Arab Emirates", "AE", "false", "false"},
						{"United Kingdom", "GB", "false", "false"},
						{"Uruguay", "UY", "false", "false"},
						{"Uzbekistan", "UZ", "false", "false"},
						{"Vanuatu", "VU", "false", "false"},
						{"Venezuela, Bolivarian Republic of", "VE", "false", "false"},
						{"Viet Nam", "VN", "false", "false"},
						{"Virgin Islands, British", "VG", "false", "false"},
						{"Virgin Islands, U.S.", "VI", "false", "false"},
						{"Wallis and Futuna", "WF", "false", "false"},
						{"Western Sahara", "EH", "false", "false"},
						{"Yemen", "YE", "false", "false"},
						{"Zambia", "ZM", "false", "false"},
						{"Zimbabwe", "ZW", "false", "false"},
					})
				},
			),
		),
		gforms.NewTextField(
			"gender",
			gforms.Validators{
				gforms.Required(),
			},
			gforms.SelectWidget(
				map[string]string{},
				func() gforms.SelectOptions {
					return gforms.StringSelectOptions([][]string{
						{"-",      "-", "false", "false"},
						{"Male",   "M", "false", "false"},
						{"Female", "F", "false", "false"},
						{"Other",  "O", "false", "false"},
					})
				},
			),
		),
		gforms.NewTextField(
			"party",
			gforms.Validators{
				gforms.Required(),
			},
			gforms.SelectWidget(
				map[string]string{},
				func() gforms.SelectOptions {
					return gforms.StringSelectOptions([][]string{
						{"-",           "-", "false", "false"},
						{"Republican",  "R", "false", "false"},
						{"Democrat",	"D", "false", "false"},
						{"Independent", "I", "false", "false"},
					})
				},
			),
		),
		gforms.NewTextField(
			"race / ethnicity",
			gforms.Validators{
				gforms.Required(),
			},
			gforms.CheckboxMultipleWidget(
				map[string]string{},
				func() gforms.CheckboxOptions {
					return gforms.StringCheckboxOptions([][]string{
						{"Hispanic, Latino, or Spanish",     		  "H", "false", "false"},
						{"American Indian or Alaska Native", 		  "I", "false", "false"},
						{"Asian", 							 		  "A", "false", "false"},
						{"Black or African American", 		 		  "B", "false", "false"},
						{"Native Hawaiian or Other Pacific Islander", "P", "false", "false"},
						{"White",									  "W", "false", "false"},
						{"Other",									  "O", "false", "false"},
					})
				},
			),
		),
		gforms.NewTextField(
			"marital status",
			gforms.Validators{
				gforms.Required(),
			},
			gforms.SelectWidget(
				map[string]string{},
				func() gforms.SelectOptions {
					return gforms.StringSelectOptions([][]string{
						{"-",                               "-", "false", "false"},
						{"Single (Never Married)",    	 	"S", "false", "false"},
						{"Divorced or Separated", 			"D", "false", "false"},
						{"Widowed", 						"W", "false", "false"},
						{"Married or Domestic Partnership",	"M", "false", "false"},
					})
				},
			),
		),		
		gforms.NewTextField(
			"furthest schooling completed",
			gforms.Validators{
				gforms.Required(),
			},
			gforms.SelectWidget(
				map[string]string{},
				func() gforms.SelectOptions {
					return gforms.StringSelectOptions([][]string{
						{"-",                                "-", "false", "false"},
						{"Less than a high school diploma",  "L", "false", "false"},
						{"High school degree or equivalent", "H", "false", "false"},
						{"Some college, but no degree",		 "S", "false", "false"},
						{"College graduate",				 "C", "false", "false"},
						{"Postgraduate study",				 "P", "false", "false"},
					})
				},
			),
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
		
		log.Printf("pw: %s confirm_pw: %s", 
			form.Data["password"].RawStr, 
			form.Data["confirm password"].RawStr)
		
		// Non-matching passwords
		if form.Data["password"].RawStr != form.Data["confirm password"].RawStr {
			tableForm.AdditionalError = "Passwords must match"
		} else { // Passwords match, everything is good - register the user
			form.MapTo(&loginData)
			fmt.Fprintf(w, "loginData ok: %v", loginData)
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
	
	http.HandleFunc("/",				frontPageHandler)

	http.HandleFunc("/login/",	loginHandler)
	http.HandleFunc("/forgotPassword/", forgotPasswordHandler)
	http.HandleFunc("/register/",	registerHandler)
	
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
		
	http.ListenAndServe(":8080", nil)
	
	log.Printf("Listening on http://localhost:8080...")
}   