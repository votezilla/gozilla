package main

 import (
 	"fmt"
 	"golang.org/x/oauth2"
 	"github.com/antonholmquist/jason"
 	//"github.com/golang/oauth2"
 	"net/http"
 	"strconv"
 	"strings"
 )

 type AccessToken struct {
 	Token  string
 	Expiry int64
 }

 func readHttpBody(response *http.Response) string {

 	fmt.Println("Reading body")

 	bodyBuffer := make([]byte, 5000)
 	var str string

 	count, err := response.Body.Read(bodyBuffer)

 	for ; count > 0; count, err = response.Body.Read(bodyBuffer) {

 		if err != nil {

 		}

 		str += string(bodyBuffer[:count])
 	}

 	return str

 }

 //Converts a code to an Auth_Token
 func GetAccessToken(client_id string, code string, secret string, callbackUri string) AccessToken {
 	fmt.Println("GetAccessToken")
 	//https://graph.facebook.com/oauth/access_token?client_id=YOUR_APP_ID&redirect_uri=YOUR_REDIRECT_URI&client_secret=YOUR_APP_SECRET&code=CODE_GENERATED_BY_FACEBOOK
 	response, err := http.Get("https://graph.facebook.com/oauth/access_token?client_id=" +
 		client_id + "&redirect_uri=" + callbackUri +
 		"&client_secret=" + secret + "&code=" + code)

 	if err == nil {

 		auth := readHttpBody(response)

 		var token AccessToken

 		tokenArr := strings.Split(auth, "&")

 		token.Token = strings.Split(tokenArr[0], "=")[1]
 		expireInt, err := strconv.Atoi(strings.Split(tokenArr[1], "=")[1])

 		if err == nil {
 			token.Expiry = int64(expireInt)
 		}

 		return token
 	}

 	var token AccessToken

 	return token
 }

 func fbHome(w http.ResponseWriter, r *http.Request) {
 	w.Header().Set("Content-Type", "text/html; charset=utf-8")

 	// generate loginURL
 	fbConfig := &oauth2.Config{
 		// ClientId: FBAppID(string), ClientSecret : FBSecret(string)
 		// Example - ClientId: "1234567890", ClientSecret: "red2drdff6e2321e51aedcc94e19c76ee"


 		ClientID:     "759729064806025", // change this to yours
 		ClientSecret: "",
 		RedirectURL:  "https://votezilla.news",
 		Scopes:       []string{"email", "user_birthday", "user_location", "user_about_me"},
 		Endpoint: oauth2.Endpoint{
 			AuthURL:  "https://www.facebook.com/dialog/oauth",
 			TokenURL: "https://graph.facebook.com/oauth/access_token",
 		},
 	}

 	url := fbConfig.AuthCodeURL("")

 	// Home page will display a button for login to Facebook

 	w.Write([]byte("<html><title>Golang Login Facebook Example</title> <body> <a href='" + url + "'><button>Login with Facebook!</button> </a> </body></html>"))
 }

 func fbLogin(w http.ResponseWriter, r *http.Request) {
 	// grab the code fragment

 	w.Header().Set("Content-Type", "text/html; charset=utf-8")
 	code := r.FormValue("code")

 	ClientId := "" // change this to yours
 	ClientSecret := ""
 	RedirectURL := "http://<domain name and don't forget port number if you use one>/FBLogin"

 	accessToken := GetAccessToken(ClientId, code, ClientSecret, RedirectURL)

 	response, err := http.Get("https://graph.facebook.com/me?access_token=" + accessToken.Token)

 	// handle err. You need to change this into something more robust
 	// such as redirect back to home page with error message
 	if err != nil {
 		w.Write([]byte(err.Error()))
 	}

 	str := readHttpBody(response)

 	prVal("str", str)

 	// dump out all the data
 	// w.Write([]byte(str))

 	// see https://www.socketloop.com/tutorials/golang-process-json-data-with-jason-package
 	user, _ := jason.NewObjectFromBytes([]byte(str))

 	prVal("user", user)

 	id, _ := user.GetString("id")
 	email, _ := user.GetString("email")
 	bday, _ := user.GetString("birthday")
 	fbusername, _ := user.GetString("username")

 	w.Write([]byte(fmt.Sprintf("Username %s ID is %s and birthday is %s and email is %s<br>", fbusername, id, bday, email)))

 	img := "https://graph.facebook.com/" + id + "/picture?width=180&height=180"

 	w.Write([]byte("Photo is located at " + img + "<br>"))
 	// see https://www.socketloop.com/tutorials/golang-download-file-example on how to save FB file to disk

 	w.Write([]byte("<img src='" + img + "'>"))
 }