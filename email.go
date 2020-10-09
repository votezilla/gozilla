package main

//TODO TMight be importing extraneous files here, I'm not sure.
import (
//	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
	//"io"
//	"net"
//	"net/mail"
//	"net/smtp"
	"net/http"
)

// Uses SSL/TLS Email Example
// See https://gist.github.com/chrisgillis/10888032 for original source and discussion.

const BUSINESS_EMAIL = "vtzilla@gmail.com"
//const BUSINESS_PASS = "vote22zilla"  //TODO TSecurity issue of including password in this code.

/* possible emails to use:
vzilla@gmail.com
votezilla.io@gmail.com
*/


func renderWelcomeEmail(username string) string {
	return fmt.Sprintf("Hello %s! This email is to confirm the creation of a new Votezilla account.\n\nNo further action is necessary.", username)
}

func renderDailyEmail(username, name string) string {
	// Render the email body template.
	return renderToString(
		kDailyEmail,
		struct { // Email template args
			FeaturedPoll	int
			Username		string
			Name			string
		} {
			FeaturedPoll:	777,
			Username: 		username,
			Name:			name,
		},
	)
}

// For testing welcome email.
func welcomeEmailHandler(w http.ResponseWriter, r *http.Request) {
	_, username := GetSessionInfo(w, r)

	body := renderWelcomeEmail(username)

	serveHtml(w, body)
}

// For testing daily email.
func dailyEmailHandler(w http.ResponseWriter, r *http.Request) {
	_, username := GetSessionInfo(w, r)

	body := renderDailyEmail(username, "Aaron Smith")

	serveHtml(w, body)
}

func testEmail() {
	pr("testEmail():")

	userData := GetUserData(36) // My userId on localhost. (Goes to my primary eml)

	subj := "Poll Question of the Day"
	body := renderDailyEmail(userData.Username, userData.Name)

	sendEmail(BUSINESS_EMAIL, "magicsquare15@gmail.com", "Aaron Smith", subj, body)
	sendEmail(BUSINESS_EMAIL, "alterego200@yahoo.com", "Aaron Smith", subj, body)
}

func dailyEmail() {
}

func sendAccountConfirmationEmail(eml, username string) {
	// Send confirmation email
	sendEmail(BUSINESS_EMAIL, eml, "Aaron Smith", "Votezilla Account Creation Confirmation", renderWelcomeEmail(username))
}

func sendEmail(from string, to_eml string, to_name string, subj string, body string) {
	prf("sendEmail %s %s %s %s %s", from, to_eml, to_name, subj, body)

	m := gomail.NewMessage()
	m.SetHeader("From", from)

	if to_name != "" {
		m.SetAddressHeader("To", to_eml, to_name)
	} else {
		m.SetHeader("To", to_eml)
	}

	m.SetHeader("Subject", subj)
	m.SetBody("text/html", body)

	if !flags.dryRun {
		pr("sending the email")
		d := gomail.NewDialer("smtp.gmail.com", 465, BUSINESS_EMAIL, flags.smtpPassword)

		// TODO: numEmailSent++

		// Send the email to Bob, Cora and Dan.
		if err := d.DialAndSend(m); err != nil {
			panic(err)
		}
	}
}
