package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
    "net"
)

var (
	blacklist 		map[string]bool
	whitelist 		map[string]bool

	blacklistArray	[]string
	whitelistArray	[]string
/*
	blackNetCount8	map[string]int
	blackNetCount16	map[string]int

	whiteNetCount8	map[string]int
	whiteNetCount16	map[string]int
*/
)

func readItemListFile(fileName string) (items map[string]bool, itemArray []string) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	items = map[string]bool{}
	itemArray = []string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		items[scanner.Text()] = true
		itemArray = append(itemArray, scanner.Text())
	}
	return
}

func dumpVal(label string, m map[string]int) {
	pr(label)
	for k, v := range m {
		prf("  %3s: %3d", k, v)
	}
}

/*
func analyzeNetCount(ipListArray []string) (map[string]int, map[string]int) {
	netCount8	:= map[string]int{}
	netCount16	:= map[string]int{}

	for _, ip := range ipListArray {
		//prVal("ip", ip)

		bytes := strings.Split(ip, ".")
		net8 := bytes[0]
		netCount8[net8]++

		if len(bytes) >= 2 {
			net16 := strings.Join(bytes[0:2], ".")
			netCount16[net16]++
		}
	}

	return netCount8, netCount16
}

func analyzeIPs() {
	blackNetCount8, blackNetCount16 = analyzeNetCount(blacklistArray)
	whiteNetCount8, whiteNetCount16 = analyzeNetCount(whitelistArray)

	//dumpVal("blackNetCount8",  blackNetCount8)
	//dumpVal("blackNetCount16", blackNetCount16)
	//dumpVal("whiteNetCount8",  whiteNetCount8)
	//dumpVal("whiteNetCount16", whiteNetCount16)
}
*/

// Returns true if it's a safe IP, false if it's an evil IP.
func checkBlacklist(ip string) bool {
	pr("checkIP: " + ip)

//	// Since we're only blocking individual IP's, don't need to check whitelist currently.
//	if _, found := whitelist[ip]; found {
//		return true
//	}

	if _, found := blacklist[ip]; found {
		pr("Blocking IP: " + ip + " due to blacklist!")
		return false
	}
/*
	bytes := strings.Split(ip, ".")

	prVal("bytes", bytes)

	net8 := bytes[0]
	if whiteNetCount8[net8] > 0 {
		return true
	}
	if blackNetCount8[net8] >= 5 {
		prf("Blocking IP: %s due to net8 count of %d!", ip, blackNetCount8[net8])
		return false
	}

	if len(bytes) >= 2 {
		net16 := strings.Join(bytes[0:2], ".")

		if whiteNetCount16[net16] > 0 {
			return true
		}
		if blackNetCount16[net16] >= 2 {
			prf("Blocking IP: %s due to net16 count of %d!", ip, blackNetCount16[net16])
			return false
		}
	}
*/
	return true
}

func recordBadIP(ip string) {
	prVal("recordBadIP", ip)

	blacklist[ip] = true

/*	// Keep track of bad ip in the runtime.
	bytes := strings.Split(ip, ".")

	net8 := bytes[0]
	blackNetCount8[net8]++

	if len(bytes) >= 2 {
		net16 := strings.Join(bytes[0:2], ".")
		blackNetCount16[net16]++
	}
*/
	// Write new bad ip to file.
	f, err := os.OpenFile("blacklist.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		pr("error: " + err.Error())
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("%s\n", ip)); err != nil {
		pr("error: " + err.Error())
	}
}

func reportError(errorMsg string) error {
	errorMsg =
		"Request blocked. " + errorMsg +
		" Contact the System Administrator at \"a l t e r e g o 2 0 0 @ y a h o o . c o m\" if you believe this is in error."
	pr(errorMsg)

	return errors.New(errorMsg)
}

func join(strList []string) string { return strings.Join(strList, "[,]") }

func logRequest(r *http.Request, ip, port, path, query, errorMsg string) {
	userId := GetSession(r)

	DbExec(`INSERT INTO vz.Request(Ip, Port, Method, Path, RawQuery, Language, Referer, UserId, Error)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8::bigint, $9);`,
			ip,
			port,
			r.Method,
			path,
			query,
			join(r.Header["Accept-Language"]),
			join(r.Header["Referer"]),
			userId,
			errorMsg)

	DbExec(`INSERT INTO vz.HasVisited(UserId, PathQuery)
			VALUES($1::bigint, $2)
			ON CONFLICT DO NOTHING;`,
			userId,
			path + "?" + query)

	// TODO: Inc DOS Attack counter here
}

// If this is an evil request, return false.  Otherwise, return true and log the request.
func CheckAndLogIP(r *http.Request) error {
	var errorMsg, path, query string

	ip, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		errorMsg = fmt.Sprintf("RemoteAddr: %q is not IP:port.  ", r.RemoteAddr)
	} else if !checkBlacklist(ip) {
		errorMsg = "Blocking bad ip: " + ip
	} else {
		path  = r.URL.Path
		query = r.URL.RawQuery

		// Block method=POST and path="/"
		if r.Method == "POST" && path == "/" {
			errorMsg = "Blocking non-logged-in post from " + ip
			//recordBadIP(ip) // This check seem legit, but it ends up blocking me somehow, so don't add to blacklist.
		} else {
			// Ban an IP if any request ends in .php, .cgi, .cmd.  Just search for ".???".
			length := len(path)
			if length >= 4 {
				//prVal("len(path)", length)
				fourthFromLastChar := path[length-4: length-3]
				//prVal("fourthFromLastChar", fourthFromLastChar)
				if fourthFromLastChar == "." {
					recordBadIP(ip)
					errorMsg = "Blocking script attack from " + ip + " for path " + path
				}
			}
		}
	}

	// Log the request in the background.
	go logRequest(r, ip, port, path, query, errorMsg)

	if errorMsg != "" {
		return reportError(errorMsg)
	}
	return nil // OK request
}

func init() {
	blacklist, blacklistArray = readItemListFile("blacklist.txt")
	whitelist, whitelistArray = readItemListFile("whitelist.txt")

//	analyzeIPs() // Not blocking subnets to be safe, just individual IP's, so we'll skip this for now.
}


/*
	//	// DISABLE THIS - it results in you not being able to log in!!!	// Block method=POST if not logged in
	//	if userId < 0 && r.Method == "POST" {
	//		pr("blocking non-logged-in post from: " + ip)
	//		recordBadIP(ip)
	//		return false
	//	}

	// Add the request string
	pr("===========================================")
	pr("logIP")

	ip, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		prf("userip: %q is not IP:port", r.RemoteAddr)
	}
	prVal("IP", ip)
	prVal("Port", port)

	prVal("Method", r.Method)	// GET
	prVal("Path", r.URL.Path)			// /article/?postId=17653&addOption=1
	prVal("RawQuery", r.URL.RawQuery)

	prVal("Host",		r.Host)

	prVal("Language", 	join(r.Header["Accept-Language"]))
	prVal("Referer", 	join(r.Header["Referer"]))
	prVal("UserAgent", 	join(r.Header["User-Agent"]))

	//prVal("r.Form.Encode()", 	r.Form.Encode())

	userId := GetSession(r)
	prVal("userId", userId)
	pr("<<")
*/
