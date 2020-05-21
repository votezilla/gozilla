package main

//TODO TMight be importing extraneous files here, I'm not sure.
import (
    "fmt"
	  "log"
    "net"
    "net/mail"
	  "net/smtp"
    "crypto/tls"
)

// Uses SSL/TLS Email Example
// See https://gist.github.com/chrisgillis/10888032 for original source and discussion. 

const BUSINESS_EMAIL = "vtzilla@gmail.com"
const BUSINESS_PASS = "vote22zilla"  //TODO TSecurity issue of including password in this code.

/*
func main() {

    from := mail.Address{"", "wisety@gmail.com"}
    to   := mail.Address{"", "wisety@gmail.com"}
    subj := "Email via Go Code"
    body := "Hi, Aaron,\n\nThe email example you sent me appears to work! (Gmail allows it if you turn on the \"Access for less secure apps\" setting.)\n\n--Tyler C, sent from Go."
	
	sendEmail(from, to, subj, body)
}
*/

func generateConfEmail(user string) string {
	//TODO TMake this email message much more palatable
	return fmt.Sprintf("Hello %s! This email is to confirm the creation of a new Votezilla account.\n\nNo further action is necessary.", user)
}

func sendEmail(from string, to string, subj string, body string) {
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

    auth := smtp.PlainAuth("",BUSINESS_EMAIL, BUSINESS_PASS, host)

    // TLS config
    tlsconfig := &tls.Config {
        InsecureSkipVerify: true,
        ServerName: host,
    }

    // Here is the key, you need to call tls.Dial instead of smtp.Dial
    // for smtp servers running on 465 that require an ssl connection
    // from the very beginning (no starttls)
    conn, err := tls.Dial("tcp", servername, tlsconfig)
    if err != nil {
        log.Panic(err)
    }

    c, err := smtp.NewClient(conn, host)
    if err != nil {
        log.Panic(err)
    }

    // Auth
    if err = c.Auth(auth); err != nil {
        log.Panic(err)
    }

    // To && From
    if err = c.Mail(fromAdr.Address); err != nil {
        log.Panic(err)
    }

    if err = c.Rcpt(toAdr.Address); err != nil {
        log.Panic(err)
    }

    // Data
    w, err := c.Data()
    if err != nil {
        log.Panic(err)
    }

    _, err = w.Write([]byte(message))
    if err != nil {
        log.Panic(err)
    }

    err = w.Close()
    if err != nil {
        log.Panic(err)
    }

    c.Quit()

}