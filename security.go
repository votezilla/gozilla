// gozilla.go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
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
	kUserId = "UserId"
)

func packInt256(buffer []byte) (q int256) {
	err := binary.Read(bytes.NewBuffer(buffer[:]), binary.LittleEndian, &q)
	check(err)	
	return
}    	

// Hashes password with salt into [4]int64 and returns result.
func GetPasswordHash256(password, salt string) int256 {
	passwordHash := sha256.Sum256([]byte(password + dbSalt))
	return packInt256(passwordHash[:])
}

// Sets a secure browser cookie.
func setCookie(w http.ResponseWriter, r *http.Request, name string, value string, expiration time.Time, encrypt bool) {
	fmt.Printf("setCookie %s -> %s\n", name, value)
	
	if encrypt {
		encrypted, err := cookieCypher.Encode(name, value)
		value = encrypted
		check(err)
		fmt.Printf("setCookie encrypting to %s\n", value)
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

	printVal("setCookie - cookie", cookie)

	http.SetCookie(w, &cookie)
}

// Gets a secure cookie
func getCookie(r *http.Request, name string, encrypted bool) (string, error) {
	print("getCookie")
	cookie, err := r.Cookie(name)
	if err != nil { // likely ErrNoCookie
		return "", err
	}
	
	if encrypted {
		var decoded string
		fmt.Printf("getCookie - decoding cookie name:'%s' cookie.Value:'%s'",
			name,
			cookie.Value)
		err := cookieCypher.Decode(name, cookie.Value, &decoded)
		check(err)
		return decoded, err
	} else {
		return cookie.Value, nil
	}
}

func longExpiration()  time.Time { return time.Now().Add(365 * 24 * time.Hour) }
func shortExpiration() time.Time { return time.Now().Add(10 * time.Minute) }

// Creates a secure session for Username.
// If RememberMe, remember cookie forever.  Otherwise, terminate cookie when browser closes.
// Returns true if successful.
// Panics on error.
func CreateSession(w http.ResponseWriter, r *http.Request, userId int, rememberMe bool) {
	// Set expiration time
	var expiration time.Time
	if rememberMe {
		expiration = longExpiration()
	} else {
		expiration = shortExpiration()
	}
	
	setCookie(w, r, kUserId, strconv.Itoa(userId), expiration, true)
}

// Returns the User.Id if the secure cookie exists, ok=false otherwise.
func GetSessionUserId(r *http.Request) (id int, ok bool) {
	print("\nGetSessionUserId")
	
	cookie, err := getCookie(r, kUserId, true)
	if err != nil { // missing or forged cookie
		fmt.Println("Cookie dos not exist", err)
		return -1, false
	}
	
	fmt.Printf("cookie: '%s'\n", cookie)
	
	userId, err := strconv.Atoi(cookie)
	
	if err != nil {
		fmt.Println("Cannot parse cookie", err)
		return -1, false
	}
	
	fmt.Printf("useId: %d\n", userId) 
	return userId, true
}

func RefreshSession(w http.ResponseWriter, r *http.Request) {
	// Get and set the cookie.  It's already encrypted.  Don't bother decrypting and re-encrypting it.
	
	cookie, err := getCookie(r, kUserId, false)
	if err != nil { // likely ErrNoCookie
		return
	}
	
	// Refresh by extending the expiration by 10 minutes.
	setCookie(w, r, kUserId, cookie, shortExpiration(), false)
}

// Should be called from init()
func InitSecurity(hashKey, blockKey string) {
	cookieCypher = securecookie.New([]byte(hashKey), []byte(blockKey))	
}