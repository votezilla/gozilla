// gozilla.go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/gorilla/securecookie"
	"net/http"
	"strconv"
	"time"
)

type int256 [4]int64

var (
	cookieCypher *securecookie.SecureCookie
)

const (
	kUserId		= "UserId"
	kRememberMe	= "RememberMe"
)

func packInt256(buffer []byte) (q int256) {
	err := binary.Read(bytes.NewBuffer(buffer[:]), binary.LittleEndian, &q)
	check(err)	
	return
}    	

// Hashes password with salt into [4]int64 and returns result.
func GetPasswordHash256(password string) int256 {
	passwordHash := sha256.Sum256([]byte(password + flags.dbSalt))
	return packInt256(passwordHash[:])
}

// Sets a secure browser cookie.
func setCookie(w http.ResponseWriter, r *http.Request, name string, value string, expiration time.Time, encrypt bool) {
	if encrypt {
		encrypted, err := cookieCypher.Encode(name, value)
		value = encrypted
		check(err)
	}

	cookie := http.Cookie {
		Name	: name,
		Value	: value,
		Expires	: expiration,
		Domain	: r.URL.Host,
		Secure	: false, // TODO: set cookie.Secure to true once SSL is enabled.
		HttpOnly: true,  // Prevent XSFR attacks. 
		Path	: "/",
	}
	
	prVal(sc_, "setCookie", cookie)
	prVal(sc_, "  expiration", expiration.Format(time.UnixDate))

	http.SetCookie(w, &cookie)
}

// Gets a secure cookie value.
func getCookie(r *http.Request, name string, encrypted bool) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil { // likely ErrNoCookie
		return "", err
	}
	
	if encrypted {
		var decoded string
		err := cookieCypher.Decode(name, cookie.Value, &decoded)
		if err != nil {
			return "", err
		}
		return decoded, err
	} else {
		return cookie.Value, nil
	}
}

// Refreshes a cookie by potentially extending its expiration.
func refreshCookie(w http.ResponseWriter, r *http.Request, name string, expiration time.Time) {
	pr(sc_, "RefreshCookie")
	
	cookieVal, err := getCookie(r, name, false)
	
	prVal(sc_, "  cookieVal", cookieVal)
	prVal(sc_, "  err", err)
	
	if err != nil { // likely ErrNoCookie
		pr(sc_, "  No cookie found to refresh")
		setCookie(w, r, name, cookieVal, expiration, false)
	}
	
	prVal(sc_, "  Setting expiration to", expiration.Format(time.UnixDate))
	
	cookie, err := r.Cookie(name)
	if err != nil { // likely ErrNoCookie
		return
	}
	
		
	cookie.Expires = expiration
	http.SetCookie(w, cookie)
}

func longExpiration()  time.Time { return time.Now().Add(365 * 24 * time.Hour) }
func shortExpiration() time.Time { return time.Now().Add(10 * time.Minute) }

// Creates a secure session for Username.
// If RememberMe, remember cookie for one year.  Otherwise, terminate cookie when browser closes.
func CreateSession(w http.ResponseWriter, r *http.Request, userId int, rememberMe bool) {
	// Set expiration time
	var expiration time.Time
	if rememberMe {
		expiration = longExpiration()
	} else {
		expiration = shortExpiration()
	}
	
	prVal(sc_, "CreateSession userId", userId)
	prVal(sc_, "              rememberMe", rememberMe)
	prVal(sc_, "              expiration", expiration.Format(time.UnixDate))
	
	setCookie(w, r, kUserId, strconv.Itoa(userId), expiration, true)
	if rememberMe {
		setCookie(w, r, kRememberMe, "true", expiration, false)
	} else {
		setCookie(w, r, kRememberMe, "false", expiration, false)
	}
}

func DestroySession(w http.ResponseWriter, r *http.Request) {
	pr(sc_, "DestroySession")
	
	setCookie(w, r, kUserId, "", time.Now(), false)
	setCookie(w, r, kRememberMe, "", time.Now(), false)
}

// Returns the User.Id if the secure cookie exists, ok=false otherwise.
func GetSession(r *http.Request) (userId int) {
	// Get userId.
	cookie, err := getCookie(r, kUserId, true)
	if err != nil { // Missing or forged cookie
		return -1
	}
	userId, err = strconv.Atoi(cookie)
	if err != nil { // Cannot parse cookie
		return -1
	}
	
	return userId
}

// Get username from userId.
func getUsername(userId int) string {		
	username := ""
	if userId != -1 {
		rows := DbQuery("SELECT Username FROM votezilla.User WHERE Id = $1;", userId)
		if rows.Next() {
			err := rows.Scan(&username)
			check(err)	
		}
		check(rows.Err())
		rows.Close()
	}
	
	return username
}

func RefreshSession(w http.ResponseWriter, r *http.Request) {
	pr(sc_, "RefreshSession")
	
	rememberMeCookie, _ := getCookie(r, kRememberMe, false)
	prVal(sc_, "  rememberMeCookie", rememberMeCookie)
	
	if rememberMeCookie == "true" {
		pr(sc_, "  rememberMeCookie == true")
	} else {
		pr(sc_, "  rememberMeCookie == false")
	}

	if rememberMeCookie == "true" {
		pr(sc_, "  RememberMe = true!")
		return // Don't refresh the cookie if rememberMe is set - it lasts for a year.
	}

	
	// Refresh by extending the expiration by 10 minutes.
	refreshCookie(w, r, kUserId, shortExpiration())
}

// Should be called from init()
func InitSecurity() {
	cookieCypher = securecookie.New([]byte(flags.secureCookieHashKey), []byte(flags.secureCookieBlockKey))	
}