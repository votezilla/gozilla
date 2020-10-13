package main

//TODO TMight be importing extraneous files here, I'm not sure.
import (
	"database/sql"
	"fmt"
	"gopkg.in/gomail.v2"
	"net/http"
	"net/url"
	"strings"
)

// Uses SSL/TLS Email Example
// See https://gist.github.com/chrisgillis/10888032 for original source and discussion.

const BUSINESS_NAME = "Votezilla"
const BUSINESS_EMAIL = "vtzilla@gmail.com"
//const BUSINESS_PASS = "vote22zilla"  //TODO TSecurity issue of including password in this code.

/* possible emails to use:
vzilla@gmail.com
votezilla.io@gmail.com
*/

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

func renderDailyEmail(email string, featuredArticleId int64) string {
	pr("renderDailyEmail")
	prVal("  featuredArticleId", featuredArticleId)
	assert(featuredArticleId >= 0)

	polls := fetchPolls(-1, 1) // Only one poll, but we need to dereference it
	assertMsg(len(polls) >= 1, "Poll not found")
	featuredArticle := polls[0]
	//featuredArticle, err := fetchArticle(featuredArticleId, -1)
	//check(err)

	//autoLoginSuffix := "?autoLoginEmail=" + url.QueryEscape(email)
	//prVal("  autoLoginSuffix", autoLoginSuffix)

	pr("  makeUrlsAbsolute")
	makeUrlsAbsolute(&featuredArticle)
	//featuredArticle.Url += autoLoginSuffix
	featuredArticle.Url = insertUrlParam(featuredArticle.Url, "autoLoginEmail", email)

	unsubscribeLink := insertUrlParam("http://votezilla.news/emailPreference/", "autoLoginEmail", email)
	prVal("  unsubscribeLink", unsubscribeLink)

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

	featuredArticleId := int64(flags.featuredArticleId) // 29825
	assert(featuredArticleId >= 0)

	body := renderDailyEmail("magicsquare15@gmail.com", featuredArticleId)

	serveHtml(w, body)
}

func testEmail() {
	pr("testEmail():")

	userData := GetUserData(36) // My userId on localhost. (Goes to my primary eml)

/*
	subj := "Welcome to Votezilla!"
	body := renderWelcomeEmail("magicsquare15@gmail.com", "magicsquare15") //
*/
	subj := "Poll Question of the Day"
	featuredArticleId := int64(flags.featuredArticleId) // 29825
	assert(featuredArticleId >= 0)
	body := renderDailyEmail("magicsquare15@gmail.com", featuredArticleId)

	sendEmail("magicsquare15@gmail.com", userData.Name, subj, body)
	sendEmail("alterego200@yahoo.com", userData.Name, subj, body)
}

// Code should work, but:
// 1) Test it with -dryRun=true
// 2) Make db record to flag false email accounts, so I don't send email to
//        them and have it bounce!  (Might mar my gmail rating.)
// 2b)Do on server too!
// 3) Make it not have to reconnect every time (see library documentation)
// 4) Look for any other goodies in the documentation.
func dailyEmail() {
	pr("dailyEmail")

	subj := "Poll Question of the Day"
	body := renderDailyEmail("magicsquare15@gmail.com", int64(flags.featuredArticleId))


	// Send each subscriber a daily emil (who should receive one).
	numSent := 0
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
			var email, name string

			err := rows.Scan(&email, &name)
			check(err)

			sendEmail(email, name, subj, body)
			numSent++
		},
		query,
	)

	prf("Daily email sent %d emails!", numSent)
}

// Send confirmation email
func sendAccountConfirmationEmail(eml, username string) {
	sendEmail(eml, "", "Welcome to Votezilla!", renderWelcomeEmail(eml, username))
}

func sendEmail(to_eml string, to_name string, subj string, body string) {
	prf("sendEmail %s %s %s %s", to_eml, to_name, subj, ellipsify(strings.Replace(body, "\r\n", " ", -1), 30))
	//prf("sendEmail %s %s %s %s", to_eml, to_name, subj, ellipsify(body, 100))

	m := gomail.NewMessage()
	m.SetAddressHeader("From", BUSINESS_EMAIL, BUSINESS_NAME)
	//m.SetHeader("From", BUSINESS_EMAIL)

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

		// TODO: numEmailSent++

		// Send the email to Bob, Cora and Dan.
		if err := d.DialAndSend(m); err != nil {
			panic(err)
		}
	}
}
