package main

//TODO TMight be importing extraneous files here, I'm not sure.
import (
	"database/sql"
	"fmt"
	"gopkg.in/gomail.v2"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Uses SSL/TLS Email Example
// See https://gist.github.com/chrisgillis/10888032 for original source and discussion.

const BUSINESS_NAME = "Votezilla"
const BUSINESS_EMAIL = "vtzilla@gmail.com" // Other possible accounts: vzilla@gmail.com, votezilla.io@gmail.com

type EmailRecipient struct {
	Email	string
	Name	string
}

func renderWelcomeEmail(email, username string) string {
	return fmt.Sprintf(
		`Hello %s,
		<br><br>
		This email is to confirm the creation of a new Votezilla account.  You're all ready to go!
		<br><br>
		Thanks for joining us in the fight to fix politics.  Here's how you can help:
		<br><br>
		1) Hop on the <a href="https://votezilla.news/?autoLoginEmail=%s">Votezilla</a> website - vote on polls, start discussions, and stay informed with balanced news.  All of these things are effective tools in making real change happen.
		<br>
		2) Join our <a href="https://www.facebook.com/groups/2402226416745938/">Facebook</a> group.
		<br>
		3) Get out there and <a href="https://www.usa.gov/how-to-vote">vote!</a>
		<br><br>
		Sincerely,
		<br><br>
		Aaron Smith
		`,
		username,
		url.QueryEscape(email))
}

func renderDailyEmail(email string) string {
	pr("renderDailyEmail")

	featuredArticleId := int64(flags.featuredArticleId)
	prVal("  featuredArticleId", featuredArticleId)
	assert(featuredArticleId >= 0)

	featuredArticle, err := fetchArticle(featuredArticleId, -1) // Only one poll, but we need to dereference it
	check(err)

	pr("  makeUrlsAbsolute")
	makeUrlsAbsolute(&featuredArticle)

	unsubscribeLink := ""

	if flags.newSubs == "" { // Don't do auto-login for newSubs, who aren't yet members.
		featuredArticle.Url = insertUrlParam(featuredArticle.Url, "autoLoginEmail", email)

		unsubscribeLink = insertUrlParam("http://votezilla.news/emailPreference/", "autoLoginEmail", email)
		prVal("  unsubscribeLink", unsubscribeLink)
	}

	// Render the email body template.
	return renderToString(
		kDailyEmail,
		struct { // Email template args
			FeaturedArticle		Article
			UnsubscribeLink		string
		} {
			FeaturedArticle:	featuredArticle,
			UnsubscribeLink:	unsubscribeLink,
		},
	)
}

// For testing welcome email.
func welcomeEmailHandler(w http.ResponseWriter, r *http.Request) {
	userId, username := GetSessionInfo(w, r)
	userData := GetUserData(userId)

	body := renderWelcomeEmail(userData.Email, username)

	serveHtml(w, body)
}

// For testing daily email.
func dailyEmailHandler(w http.ResponseWriter, r *http.Request) {
	pr("dailyEmailHandler")

	featuredArticleId := int64(flags.featuredArticleId)
	assert(featuredArticleId >= 0)

	body := renderDailyEmail(flags.testEmailAddress)

	serveHtml(w, body)
}

/*
func testEmail() {
	pr("testEmail():")

	subj := "Welcome to Votezilla!"
	body := renderWelcomeEmail(flags.testEmailAddress, flags.testUsername) //

	subj := "Poll Question of the Day"
	body := renderDailyEmail(flags.testEmailAddress)

	sendEmail(flags.testEmailAddress, flags.tes
}*/

// Code should work, but:
// 1) Test it with -dryRun=true
// 2) Make db record to flag false email accounts, so I don't send email to
//        them and have it bounce!  (Might mar my gmail rating.)
// 2b)Do on server too!
// 3) Make it not have to reconnect every time (see library documentation)
// 4) Look for any other goodies in the documentation.
func dailyEmail() {
	pr("dailyEmail")

	recipients := make([]EmailRecipient, 0)
	if flags.newSubs != "" { // For newsletter subscribers, who are not votezilla members yet.
		assert(flags.newSubs != "")
		assert(flags.emailTarget == "newSubs")

		// Get list of users emails, so we can exclude them and not send double daily emails.
		excludeEmails := make(map[string]bool)
		DoQuery(
			func(rows *sql.Rows) {
				var email string

				err := rows.Scan(&email)
				check(err)

				excludeEmails[email] = true
			},
			`SELECT Email FROM $$User`,
		)
		prVal("excludeEmails", excludeEmails)

		// Parse the newSubs (new subscribers) from the commandline, separate by comma delimiter.
		emails := strings.Split(flags.newSubs, (","))

		for _, email := range emails {
			_, found := excludeEmails[email]
			if !found {
				prVal("Adding recipient", email)
				recipient := EmailRecipient{Email: email}

				recipients = append(recipients, recipient)
			} else {
				prVal("Excluding recipient", email)
			}
		}
	} else {
		// Send each subscriber a daily emil (who should receive one).
		{
			query := `SELECT Email, COALESCE(Name, '') FROM $$User WHERE NOT FakeEmail`
			switch strings.ToLower(flags.emailTarget) {
				case "daily":
					query = query + ` AND COALESCE(EmailPreference, '') IN ('', 'Daily', 'Test')`
					break;
				case "test":
					query = query + ` AND COALESCE(EmailPreference, '') = 'Test'`
					break;
				default:
					panic("  Unhandled flags.emailTarget: '" + flags.emailTarget + "'")
			}
			DoQuery(
				func(rows *sql.Rows) {
					recipient := EmailRecipient{}

					err := rows.Scan(&recipient.Email, &recipient.Name)
					check(err)

					recipients = append(recipients, recipient)
				},
				query,
			)
		}
	}
	prVal("recipients", recipients)

	sendBulkEmail(
		recipients,
		ternary_str(flags.emailSubject != "", flags.emailSubject, "Poll Question of the Day"),
		renderDailyEmail,
	)
}

// Send confirmation email.
func sendAccountConfirmationEmail(eml, username string) {
	sendEmail(eml, "", "Welcome to Votezilla!", renderWelcomeEmail(eml, username))
}

// Send a single email.
func sendEmail(to_eml string, to_name string, subj string, body string) {
	prf("sendEmail %s %s %s %s", to_eml, to_name, subj, ellipsify(strings.Replace(body, "\r\n", " ", -1), 30))

	m := gomail.NewMessage()
	m.SetAddressHeader("From", BUSINESS_EMAIL, BUSINESS_NAME)

	if to_name != "" {
		m.SetAddressHeader("To", to_eml, to_name)
	} else {
		m.SetHeader("To", to_eml)
	}

	m.SetHeader("Subject", subj)
	m.SetBody("text/html", body)

	if flags.dryRun {
		//prf("  dryRun; would have sent email to %s %s", to_name, to_eml)
	} else {
		pr("sending the email")
		d := gomail.NewDialer("smtp.gmail.com", 465, BUSINESS_EMAIL, flags.smtpPassword)

		// Send the email to Bob, Cora and Dan.
		if err := d.DialAndSend(m); err != nil {
			panic(err)
		}
	}
}

// Sends a bulk email blast (newsletter) to a group of email recipients (subscribers).
func sendBulkEmail(recipients []EmailRecipient, subj string, emailRenderer func(email string)string) {
	prf("sendBulkEmail recipients(%d) %s", len(recipients), subj)

	numSent := 0

	// Connect to the SMTP server.
	var s gomail.SendCloser
	var err error
	if !flags.dryRun {
		d := gomail.NewDialer("smtp.gmail.com", 465, BUSINESS_EMAIL, flags.smtpPassword)
		s, err = d.Dial()
		check(err)
	}

	// Send each message.
	m := gomail.NewMessage()
	for _, recipient := range(recipients) {
		to_eml := recipient.Email
		to_name := recipient.Name
		body := emailRenderer(to_eml)

		prf("  sendEmail %s %s %s %s", to_eml, to_name, subj, ellipsify(strings.Replace(body, "\r\n", " ", -1), 30))

		m.SetAddressHeader("From", BUSINESS_EMAIL, BUSINESS_NAME)
		if to_name != "" {
			m.SetAddressHeader("To", to_eml, to_name)
		} else {
			m.SetHeader("To", to_eml)
		}
		m.SetHeader("Subject", subj)
		m.SetBody("text/html", body)

		if flags.dryRun {
			//prf("  dryRun; would have sent email to %s %s", to_name, to_eml)
		} else {
			pr("sending the email")

			// Send the email to Bob, Cora and Dan.
			check(gomail.Send(s, m))

			numSent++
		}
		m.Reset()

		if numSent % 3 == 0 {
			prf("  Just sent %d emails; waiting 3 minutes.", numSent)

			if !flags.dryRun {
				time.Sleep(3 * time.Minute)
			}
		}
	}

	prf("  BULK EMAIL FINISHED - SENT %d MESSAGES! (dryRun = %s)", numSent, bool_to_str(flags.dryRun))
}



///////////////////////////////////////////////////////////////////////////////
//
// import and export subscribers
//
///////////////////////////////////////////////////////////////////////////////
func exportSubsHandler(w http.ResponseWriter, r *http.Request){
	// TODO: assert(userId == 5)

	pr("exportSubsHandler")

	tr := func(s string) string { return "<tr>" + s + "</tr>" }
	td := func(s string) string { return "<td>" + s + "</td>" }

	table := "<table>"
	table = table + tr(td("email") + td("name") + td("first name") + td("last name"))
	DoQuery(
		func(rows *sql.Rows) {
			var email, name string

			err := rows.Scan(&email, &name)
			check(err)

			names := strings.Split(name, " ")

			prVal("name", name)
			prVal("names", names)

			var firstName, lastName string

			if len(names) > 0 {
				firstName = names[0]
				lastName = names[len(names)-1]
			}

			table = table + tr(td(email) + td(name) + td(firstName) + td(lastName))

		},
		//"SELECT Email, COALESCE(Name, '') FROM $$User")
		"SELECT Email, COALESCE(Name, '') FROM $$User WHERE NOT FakeEmail")
	table = table + "</table>"

	serveHtml(w, table)
}

func importSubsHandler(w http.ResponseWriter, r *http.Request){

}

