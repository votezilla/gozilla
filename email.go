package main

//TODO TMight be importing extraneous files here, I'm not sure.
import (
	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
	//"io"
	"net"
	"net/mail"
	"net/smtp"
)

// Uses SSL/TLS Email Example
// See https://gist.github.com/chrisgillis/10888032 for original source and discussion.

const BUSINESS_EMAIL = "vtzilla@gmail.com"
//const BUSINESS_PASS = "vote22zilla"  //TODO TSecurity issue of including password in this code.

/* possible emails to use:
vzilla@gmail.com
votezilla.io@gmail.com
*/


func testEmail() {
	pr("testEmail():")

    from := BUSINESS_EMAIL
    //to   := "magicsquare15@gmail.com"
    //subj := "Email via Go Code"
    //body := "Hi, Aaron,\n\nThe email example you sent me appears to work! (Gmail allows it if you turn on the \"Access for less secure apps\" setting.)\n\n--Tyler C, sent from Go."

	//sendEmail(from, to, subj, body)
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", "magicsquare15@gmail.com", "alterego200@yahoo.com")
	m.SetAddressHeader("Cc", "sheeple.game@gmail.com", "Sheepie")
	m.SetHeader("Subject", "Hello!")
//	m.SetBody("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")
//	m.Attach("static\\ballotboxes\\ballotbox 3dinos small 160x180.jpg") // works.  would need '/' for linux.

	// Render the email body template.
	body := renderToString(
		kDailyEmail,
		struct { // Email template args
			FeaturedPoll	int
		} {
			FeaturedPoll : 777,
		},
	)
	prVal("body", body)

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

/*
func InitEmail() {
	d := gomail.NewDialer("smtp.example.com", 587, "user", "123456")
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
}*/


func generateConfEmail(user string) string {
	//TODO TMake this email message much more palatable
	return fmt.Sprintf("Hello %s! This email is to confirm the creation of a new Votezilla account.\n\nNo further action is necessary.", user)
}

func sendEmail(from string, to string, subj string, body string) {
    //pr("sendEmail: DISABLING EMAIL UNTIL FIXED")

	// TODO: replace checks with an exception, make it not accept the email address if that is what was failing when registering.

	//return;

	prf("sendEmail %s %s %s %s", from, to, subj, body)

	//TODO TValidate address strings here
	//(Not sure why the example was wrapping the strings, here, but I left it be, mechanically.)
    fromAdr := mail.Address{"", from}
    toAdr   := mail.Address{"", to}

    // Setup headers
    headers := make(map[string]string)
    headers["From"] = fromAdr.String()
    headers["To"] = toAdr.String()
    headers["Subject"] = subj

    // Setup message
    message := ""
    for k,v := range headers {
        message += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    message += "\r\n" + body

    // Connect to the SMTP Server
    servername := "smtp.gmail.com:465"

    host, _, _ := net.SplitHostPort(servername)

    //prVal("flags.smtpPassword", flags.smtpPassword)

    auth := smtp.PlainAuth("",BUSINESS_EMAIL, flags.smtpPassword, host)

    // TLS config
    tlsconfig := &tls.Config {
        InsecureSkipVerify: true,
        ServerName: host,
    }

    // Here is the key, you need to call tls.Dial instead of smtp.Dial
    // for smtp servers running on 465 that require an ssl connection
    // from the very beginning (no starttls)
    conn, err := tls.Dial("tcp", servername, tlsconfig)
    check(err)

    c, err := smtp.NewClient(conn, host)
    check(err)

    // Auth
    check(c.Auth(auth))

    // To && From
    check(c.Mail(fromAdr.Address))

    check(c.Rcpt(toAdr.Address))

    // Data
    w, err := c.Data()
    check(err)

    _, err = w.Write([]byte(message))
    check(err)

    check(w.Close())

    c.Quit()

	//pr("Done, exiting.")
}
