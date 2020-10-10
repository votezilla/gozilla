// gozilla.go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/gorilla/securecookie"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type int256 [4]int64

type UserData struct {
	Email		string
	Username	string
	Name		string
}

var (
	cookieCypher *securecookie.SecureCookie
)

const (
	kUserId		= "UserId"
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
		HttpOnly: true,  // Prevents XSFR attacks, but does not allow server-side cookies to be read on the client side via JS.
		Path	: "/",
		SameSite: http.SameSiteLaxMode,
	}

	prVal("setCookie", cookie)
	prVal("  expiration", expiration.Format(time.UnixDate))

	http.SetCookie(w, &cookie)
}

// Gets a secure cookie value.
func getCookie(r *http.Request, name string, encrypted bool) (string, error) {
	//prVal("getCookie ", name)
	cookie, err := r.Cookie(name)

	//prVal("  r.Cookie", cookie)

	if err != nil {  // likely ErrNoCookie
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

// Sets a cookie the easy way.
func SetCookie(w http.ResponseWriter, r *http.Request, name string, value string) {
	setCookie(w, r, name, value, longExpiration(), false)
}

// Gets a cookie the easy way.
func GetCookie(r *http.Request, name string, defaultValue string) string {
	value, err := getCookie(r, name, false)
	if err != nil {
		return defaultValue  // likely ErrNoCookie
	} else {
		return value
	}
}

func GetAndDecodeCookie(r *http.Request, name string, defaultValue string) string {
	cookieVal := GetCookie(r, name, defaultValue)

	prVal("encoded cookieVal", cookieVal)

	// decode return address
	decodedCookieVal, err := url.QueryUnescape(cookieVal)
	check(err)

	prVal("decoded cookieVal", cookieVal)

	return decodedCookieVal
}

// Refreshes a cookie by potentially extending its expiration.
func refreshCookie(w http.ResponseWriter, r *http.Request, name string, expiration time.Time) {
/*	prVal("RefreshCookie", name)

	cookieVal, err := getCookie(r, name, false)

	prVal("  cookieVal", cookieVal)
	prVal("  err", err)

	if err != nil { // likely ErrNoCookie
		pr("  No cookie found to refresh")
		setCookie(w, r, name, cookieVal, expiration, false)
	}

	prVal("  Setting expiration to", expiration.Format(time.UnixDate))

	cookie, err := r.Cookie(name)
	if err != nil { // likely ErrNoCookie
		return
	}


	cookie.Expires = expiration
	http.SetCookie(w, cookie)
*/
}

func longExpiration()  time.Time { return time.Now().Add(365 * 24 * time.Hour) }
func shortExpiration() time.Time { return time.Now().Add(10 * time.Minute) }

// Creates a secure session for Username.
// If RememberMe, remember cookie for one year.  Otherwise, terminate cookie when browser closes.
func CreateSession(w http.ResponseWriter, r *http.Request, userId int64) {
	setCookie(w, r, kUserId, strconv.FormatInt(userId, 10), longExpiration(), true)
}

func DestroySession(w http.ResponseWriter, r *http.Request) {
	pr("DestroySession")

	setCookie(w, r, kUserId, "", time.Now(), false)
	//setCookie(w, r, kRememberMe, "false", time.Now(), false)

	pr("Test DestroySession:")
	prVal("  getCookie(kUserId)", GetCookie(r, kUserId, "not found"))
}

// Returns the User.Id if the secure cookie exists, ok=false otherwise.
func GetSession(w http.ResponseWriter, r *http.Request) (userId int64) {
	pr("GetSession")

	// Friction-free login from email if url has &autoLoginEmail=[eml]
	escapedEmail := parseUrlParam(r, "autoLoginEmail")
	prVal("  escapedEmail", escapedEmail)
	if escapedEmail != "" {
		autoLoginEmail, err := url.QueryUnescape(escapedEmail)
		check(err)
		prVal("  autoLoginEmail", autoLoginEmail)

		userId := int64(-1)
		rows := DbQuery("SELECT Id FROM $$User WHERE Email = $1", autoLoginEmail)
		if rows.Next() {
			err := rows.Scan(&userId)
			check(err)
		}
		check(rows.Err())
		rows.Close()

		if userId >= 0 {
			prf("  XXX Auto-Login from email successful!  UserId = %d", userId)

			CreateSession(w, r, userId)

			return userId
		}
	}

	// Get userId from the session cookie.
	cookie, err := getCookie(r, kUserId, true)
	if err != nil { // Missing or forged cookie
		if flags.testUserId != "" {
			cookie = flags.testUserId

			pr("  userId set to testUserId")
		} else {
			return -1
		}
	}
	userId, err = strconv.ParseInt(cookie, 10, 64)
	if err != nil { // Cannot parse cookie
		pr("Cannot parse cookie")
		return -1
	}

	prVal("  userId", userId)

	return userId
}

// Get userId, username from the secure cookie.
func GetSessionInfo(w http.ResponseWriter, r *http.Request) (userId int64, username string) {
	pr("GetSessionInfo")
	userId = GetSession(w, r)
	prVal("  userId", userId)
	if userId == -1 {
		pr(`GetSessionInfo: -1, ""`)
		return -1, ""
	}

	username = ""
	rows := DbQuery("SELECT Username FROM $$User WHERE Id = $1::bigint;", userId)
	if rows.Next() {
		err := rows.Scan(&username)
		check(err)
	} else {  // User was deleted from db; destroy session to maintain consistency.
		DestroySession(w, r)
		return -1, ""
	}
	check(rows.Err())
	rows.Close()

	prf("GetSessionInfo %d, %s", userId, username)

	return
}

func GetUserData(userId int64) (userData UserData) {
	rows := DbQuery("SELECT Email, Username, COALESCE(Name, '') FROM $$User WHERE Id = $1::bigint;", userId)
	if rows.Next() {
		err := rows.Scan(&userData.Email, &userData.Username, &userData.Name)
		check(err)
	}
	check(rows.Err())
	rows.Close()

	prVal("userData", userData)
	return userData
}

func InvalidateCache(userId int64) {
	DbExec(`UPDATE $$User
		       SET LastModTime = Now()
		     WHERE Id = $1::bigint;`,
		    userId)
}

// Get userId, username, isCacheValid from the secure cookie & database lookup.
func GetSessionInfo2(w http.ResponseWriter, r *http.Request) (userId int64, username string, isCacheValid bool) {
	pr("GetSessionInfo2")
	userId = GetSession(w, r)
	prVal("  userId", userId)

	if userId == -1 {
		pr(`GetSessionInfo: -1, ""`)
		return -1, "", true
	}

	username = ""
	isCacheValid = false
	rows := DbQuery(`SELECT Username,
							Now() > LastModtime + interval '3 minutes'
					   FROM $$User
					  WHERE Id = $1::bigint;`,
					 userId)
	if rows.Next() {
		err := rows.Scan(&username, &isCacheValid)
		prf("  username=%s isCacheValid=%s", username, bool_to_str(isCacheValid))
		check(err)
	} else {  // User was deleted from db; destroy session to maintain consistency.
		pr("  DestroySession")
		DestroySession(w, r)
		return -1, "", true
	}
	check(rows.Err())
	rows.Close()

	prf("  GetSessionInfo2 %d, %s, %s", userId, username, bool_to_str(isCacheValid))

	return userId, username, isCacheValid
}

func GetUsername(userId int64) (username string) {
	rows := DbQuery("SELECT Username FROM $$User WHERE Id = $1::bigint;", userId)
	if rows.Next() {
		err := rows.Scan(&username)
		check(err)
	} else {  // User was deleted from db; destroy session to maintain consistency.
		panic("userId not found")
		return ""
	}
	check(rows.Err())
	rows.Close()
	return
}

// Return -1 if not found or error.
func UsernameToUserId(username string) int64 {
	prVal("UsernameToUserId... username", username)
	rows := DbQuery("SELECT Id FROM $$User WHERE Username = $1;", username)
	if rows.Next() {
		var userId int64
		err := rows.Scan(&userId)
		check(err)
		return userId
	} else {  // User was deleted from db; destroy session to maintain consistency.
		panic("userId not found")
		return -1
	}
	check(rows.Err())
	rows.Close()
	return -1
}

func RefreshSession(w http.ResponseWriter, r *http.Request) {
/*	pr("RefreshSession")

	rememberMeCookie, _ := getCookie(r, kRememberMe, false)
	prVal("  rememberMeCookie", rememberMeCookie)

	prVal("rememberMeCookie", rememberMeCookie)

	if rememberMeCookie == "true" {
		pr("  RememberMe = true!")
		return // Don't refresh the cookie if rememberMe is set - it lasts for a year.
	}

	// Refresh by extending the expiration by 10 minutes.
	refreshCookie(w, r, kUserId, shortExpiration())*/
}

// Should be called from init()
func InitSecurity() {
	cookieCypher = securecookie.New([]byte(flags.secureCookieHashKey), []byte(flags.secureCookieBlockKey))
}

