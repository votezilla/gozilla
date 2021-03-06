package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/golang-collections/go-datastructures/bitarray"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ref: https://www.ip2location.com/free/robot-whitelist
// could add these results to the whitelist:  select email, ip, userid from vz.request r join vz.user u on r.userid=u.id where userid >= 0 group by 1, 2, 3 order by 2, 1;

type IPs bitarray.BitArray

var (
	blacklist		*IPs
	whitelist		*IPs

	dosCounter map[int]int = make(map[int]int, 0)
)

func ip_to_int(ip string) int {
	val := 0

	//prVal("ip_to_int ip", ip)

	parts := strings.Split(ip, ".")
	assert(len(parts) == 4)
	for i := 0; i < 4; i++ {
		val *= 256

		iVal := str_to_int(parts[i])

		val += iVal
	}

	return val
}

func int_to_ip(ip int) string {
	parts := make([]string, 4)

	//prVal("int_to_ip ip", ip)

	pos := 3
	for ip > 0 {
		parts[pos] = int_to_str(ip % 256)

		ip >>= 8
		//prVal("  ip", ip)

		pos--
	}

	//prVal("  parts", parts)

	return strings.Join(parts, ".")
}

func registerIPSubnet(ips *IPs, ip, subnetBits int) {
	//prf("registerIPSubnet %s/%d", ip, subnetBits)
	assert(0 <= subnetBits && subnetBits <= 32)
	rangeBits := 32 - subnetBits

	numIPs := 1 << rangeBits
	for i := 0; i < numIPs; i++ {
		(*ips).SetBit(uint64(ip + i))

		//prVal("  registering IP", int_to_ip(ip + i))
	}
}

func checkIP(ips *IPs, ip int) bool {
	//prVal("checkIP", ip)

	bit, err := (*ips).GetBit(uint64(ip))
	check(err)
	return bit
}


func readIPsFile(fileName string) *IPs {
	prVal("readIPsFile", fileName)

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	//ips := IPs(bitarray.NewBitArray(1 << 32))
	ips := IPs(bitarray.NewSparseBitArray())

	//prVal("ips.Capacity()", ips.Capacity())
	//prVal("size = ", ips.Capacity() / 8)

	lineNum := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()

		//prf("line %d text %s", lineNum, text)

		tokens 		:= strings.Split(text, "/")
		//prVal("tokens", tokens)
		ip 			:= ip_to_int(tokens[0])
		subnetBits 	:= 32
		if len(tokens) == 2 {
			subnetBits = str_to_int(tokens[1])
		}

		//prf("tokens[0] %s ip %d subnetBits %d", tokens[0], ip, subnetBits)

//		ips = append(ips, createSubnetList(ip, subnetBits))

		registerIPSubnet(&ips, ip, subnetBits)

		lineNum++
	}

	//prVal("sizeof(ips)", unsafe.Sizeof(ips))

	return &ips
}

func recordBadIP(ip string) {
	prVal("recordBadIP", ip)

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

func logRequest(w http.ResponseWriter, r *http.Request, ip, port, path, query, errorMsg string) {
	userId := GetSession(w, r)

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

// Returns true if this is a DOS attack - 100 requests in a minute.
func checkDOSAttack(ip int) bool {
	count, _ := dosCounter[ip]
	count++

	if count == 100 {
		return true
	}

	dosCounter[ip] = count

	prf("Current dos count for ip %s: %d", int_to_ip(ip), count)

	return false
}
func resetDOSCounters() {
	for {
		time.Sleep(1 * time.Minute)

		dosCounter = make(map[int]int, 0)

		//pr("Resetting dos counter")
	}
}

// If this is an evil request, return false.  Otherwise, return true and log the request.
func CheckAndLogIP(w http.ResponseWriter, r *http.Request) error {
	pr("CheckAndLogIP")

	var errorMsg, path, query string

	ip, port, err := net.SplitHostPort(r.RemoteAddr)

	path  = r.URL.Path
	query = r.URL.RawQuery

	if flags.skipFirewall {
		go logRequest(w, r, ip, port, path, query, "skipping firewall")
		return nil // ok request
	}

	if err != nil {
		errorMsg = fmt.Sprintf("RemoteAddr: %q is not IP:port.  ", r.RemoteAddr)
	} else if ip == "::1" {
		// localhost - ok
	} else {
		ipVal := ip_to_int(ip)

		if checkIP(whitelist, ipVal) {
			// ok
		} else if checkIP(blacklist, ipVal) {
			errorMsg = "Blocking blacklisted ip: " + ip
		} else if checkDOSAttack(ipVal) {
			errorMsg = "Preventing DOS Attack"
			recordBadIP(ip)
		} else {
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
					prVal("fourthFromLastChar", fourthFromLastChar)

					if fourthFromLastChar == "." {
						extension := path[length-3:]

						prVal("extension", extension)

						// Block .php, .cgi, .cmd.  Make sure we don't block robots.txt !!!
						if extension == "php" || extension == "cgi" || extension == "cmd" {
							recordBadIP(ip)
							errorMsg = "Blocking script attack from " + ip + " for path " + path
						}
					}
				}
			}
		}
	}

	// Log the request in the background.
	go logRequest(w, r, ip, port, path, query, errorMsg)

	if errorMsg != "" {
		return reportError(errorMsg)
	}
	return nil // OK request
}

func InitFirewall() {
	if flags.skipFirewall {
		return
	}

	pr("reading blacklist")
	blacklist = readIPsFile("blacklist.txt")
	emptyList := IPs(bitarray.NewSparseBitArray())
	whitelist = &emptyList
	if flags.skipWhitelist {
		pr("skipping whitelist")
	} else {
		pr("reading whitelist in background")
		readWhitelist := func() {
			pr("  started reading whitelist...")
			whitelist = readIPsFile("whitelist.txt")
			pr("  finished reading whitelist!!!")
		}
		go readWhitelist()
	}
	go resetDOSCounters()
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

	userId := GetSession(w, r)
	prVal("userId", userId)
	pr("<<")
*/
