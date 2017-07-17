// gozilla.go
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/gorilla/securecookie"
	"github.com/lib/pq"
	"net/http"
	"time"
)

type int256 [4]int64

var (
	cookieEncrypter *securecookie.SecureCookie
)

const (
	kSecureCookie = "SecureCookie"
)

func packInt256(buffer []byte) (q int256) {
	err := binary.Read(bytes.NewBuffer(buffer[:]), binary.LittleEndian, &q)
	check(err)	
	return
}

func unserializeInt256(str string) (q int256) {
	err := binary.Read(bytes.NewBufferString(str), binary.LittleEndian, &q)
	check(err)	
	return
}

func serializeInt256(q int256) string {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, q)
	check(err)	
	return buf.String()
}

// Returns random bytes.
func randBytes(len int) []byte {
	b := make([]byte, len)
	_, err := rand.Read(b)
	check(err)
	return b
}    	

// Hashes password with salt into [4]int64 and returns result.
func GetPasswordHash256(password, salt string) int256 {
	passwordHash := sha256.Sum256([]byte(password + dbSalt))
	return packInt256(passwordHash[:])
}

// Encode and decode int256 to/from cookie.
func encodeInt256Cookie(q int256) string {
	secureCookie := serializeInt256(q)
	fmt.Printf("secureCookie: %s\n", secureCookie)

	encoded, err := cookieEncrypter.Encode(kSecureCookie, secureCookie)
	check(err)
	fmt.Printf("encoded: %T %s\n", encoded, encoded)
	
	return encoded	
}

func decodeInt256Cookie(cookie string) int256 {
	var decoded string
	err := cookieEncrypter.Decode(kSecureCookie, cookie, &decoded)
	check(err)
	fmt.Printf("decoded: %T %s\n", decoded, decoded)
	
	p := unserializeInt256(decoded)
	fmt.Printf("p: %v\n", p)
	
	return p
}

// Sets a browser cookie.
func setCookie(w http.ResponseWriter, r *http.Request, name string, value string, expiration time.Time) {
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

func longExpiration()  time.Time { return time.Now().Add(365 * 24 * time.Hour) }
func shortExpiration() time.Time { return time.Now().Add(10 * time.Minute) }

// Creates a secure session for Username.
// If RememberMe, remember cookie forever.  Otherwise, terminate cookie when browser closes.
// Returns true if successful.
// Panics on error.
func CreateSession(w http.ResponseWriter, r *http.Request, userId int, rememberMe bool) bool {
	// Generate a random 256-bye number stored as in [4]int64 format that is unique in the Session dabase.
	for tries := 0; tries < 10; tries++ {
		// Generate 32 random bytes
		bytes := randBytes(32)
		
		printVal("CreateSession randBytes - ", bytes) 
		
		secureCookie := packInt256(bytes)
		
		printVal("CreateSession - secureCookie", secureCookie)
		
		printVal("CreateSession - pq.Array(secureCookie[1:])", pq.Array(secureCookie[1:]))
		
		if DbUnique("SELECT * FROM votezilla.Session WHERE Id = $1 AND SecureCookie = $2",
			secureCookie[0],
			pq.Array(secureCookie[1:])) {
			// Set expiration time
				var expiration time.Time
				if rememberMe {
					expiration = longExpiration()
				} else {
					expiration = shortExpiration()
			}
			
			// Set the session database entry.
			DbQuery("INSERT INTO votezilla.Session (Id, SecureCookie, UserId, Expiration) VALUES ($1, $2, $3, $4);",
				secureCookie[0],			// secureCookie - first int64 stored in Id
				pq.Array(secureCookie[1:]),	// secureCookie - last 3 int64's stored in SessionCookie
				userId,
				expiration)
				
			// Convert secureCookie to string
			secureCookieStr := encodeInt256Cookie(secureCookie)
			fmt.Printf("* encoded: %T %s\n", secureCookieStr, secureCookieStr)
				
			printVal("secureCookie", secureCookie)
			printVal("secureCookieStr", secureCookieStr)
			
			// Set the session cookie			
			setCookie(w, r, kSecureCookie, secureCookieStr, expiration)

			return true
		}
	}
	
	return false // Unable to generate unique SecureCookie after 10 attempts, giving up.
}

// Returns the User.Id if the secure cookie exists, ok=false otherwise.
func GetSessionUserId(r *http.Request) (id int, ok bool) {
	print("\nGetSessionUserId")
	
	cookie, err := r.Cookie(kSecureCookie)
	if err != nil { // likely ErrNoCookie
		fmt.Println("Cookie dos not exist", err)
		return -1, false
	}
	
	fmt.Printf("cookie.Value %T '%s'", cookie.Value, cookie.Value)
	
	fmt.Printf("scanning for cookie hexes")
	// Convert cookie string to secureCookie.
	secureCookie := decodeInt256Cookie(cookie.Value)
	fmt.Printf("* p: %T %v\n", secureCookie, secureCookie)
	
	printVal("secureCookie", secureCookie)
	
	// Lookup secureCookie in database.
	rows := DbQuery("SELECT UserId FROM votezilla.Session WHERE Id = $1 AND SecureCookie = $2;",
		secureCookie[0],			// secureCookie - first int64 packed in Id
		pq.Array(secureCookie[1:]))	// secureCookie - last 3 int64's packed in SessionCookie
	
	if (rows.Next()) {
		// Secure cookie, return sessions's UserId
		var userId int
		check(rows.Scan(&userId))
		return userId, true
	}
	
	// Session not found.
	return -1, false
}

func RefreshSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(kSecureCookie)
	if err != nil { // likely ErrNoCookie
		return
	}
	
	// Refresh by extending the expiration by 10 minutes.
	setCookie(w, r, kSecureCookie, cookie.Value, shortExpiration())
}

// Should be called from init()
func InitSecurity(hashKey, blockKey string) {
	cookieEncrypter = securecookie.New([]byte(hashKey), []byte(blockKey))	
}