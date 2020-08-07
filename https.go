package main
/*
// https://blog.kowalczyk.info/article/Jl3G/https-for-free-in-go.html
// To run:
// go run main.go
// Command-line options:
//   -production : enables HTTPS on port 443
//   -redirect-to-https : redirect HTTP to HTTTPS

import (
	"context"
	"crypto/tls"
	"path/filepath"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

const (
	htmlIndex = `<html><body>Welcome!</body></html>`
	httpPort  = "127.0.0.1:8080"
)

func makeServerFromMux(mux *http.ServeMux) *http.Server {
	// set timeouts so that a slow or malicious client doesn't
	// hold resources forever
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
}

// This is the real webserver, which calls back to main to set up the paths
func makeHTTPServer() *http.Server {
//	mux := &http.ServeMux{}
	//mux.HandleFunc("/", handleIndex)

	mux := SetupWebHandlers() // calls back to main

	return makeServerFromMux(mux)
}

// All this server does is redirect HTTP to HttpS
func makeHTTPToHTTPSRedirectServer() *http.Server {
	mux := &http.ServeMux{}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		newURI := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, newURI, http.StatusFound)
	})

	return makeServerFromMux(mux)
}


func InitWebServer() {
	//parseFlags()
	var m *autocert.Manager

	if flags.inProduction {
		hostPolicy := func(ctx context.Context, host string) error {
			// Note: change to your real host
			//allowedHost := "www.mydomain.com"
			allowedHost := "votezilla.io"
			prVal("host", host)
			if host == allowedHost {
				pr("Not allowed host!")
				return nil
			}
			return fmt.Errorf("acme/autocert: only %s host is allowed", allowedHost)
		}

		dataDir := "."
		m = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: hostPolicy,
			Cache:      autocert.DirCache(dataDir),
		}

		httpsSrv := makeHTTPServer()
		httpsSrv.Addr = ":443"
		httpsSrv.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}

		go func() {
			fmt.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
			err := httpsSrv.ListenAndServeTLS("", "")
			if err != nil {
				log.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}()
	}

	var httpSrv *http.Server
	if flags.redirectHTTPToHTTPS {
		httpSrv = makeHTTPToHTTPSRedirectServer()
	} else {
		httpSrv = makeHTTPServer()
	}
	// allow autocert handle Let's Encrypt callbacks over http
	if m != nil {
		httpSrv.Handler = m.HTTPHandler(httpSrv.Handler)
	}

	httpSrv.Addr = httpPort
	fmt.Printf("Starting HTTP server on %s\n", httpPort)

	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}

///////////////////////////////////////////////////////////////////////////////////////
// Ref: [200~https://marcofranssen.nl/build-a-go-webserver-on-http-2-using-letsencrypt/
func redirectHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
	    http.Error(w, "Use HTTPS", http.StatusBadRequest)
	    return
	}
	target := "https://" + stripPort(r.Host) + r.URL.RequestURI()
	http.Redirect(w, r, target, http.StatusFound)
}

func stripPort(hostport string) string {
  host, _, err := net.SplitHostPort(hostport)
  if err != nil {
    return hostport
  }
  return net.JoinHostPort(host, "443")
}

func getSelfSignedOrLetsEncryptCert(certManager *autocert.Manager) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	    dirCache, ok := certManager.Cache.(autocert.DirCache)
	    if !ok {
	      dirCache = "certs"
	    }

	    keyFile := filepath.Join(string(dirCache), hello.ServerName+".key")
	    crtFile := filepath.Join(string(dirCache), hello.ServerName+".crt")
	    certificate, err := tls.LoadX509KeyPair(crtFile, keyFile)
	    if err != nil {
	      fmt.Printf("%s\nFalling back to Letsencrypt\n", err)
	      return certManager.GetCertificate(hello)
	    }
	    fmt.Println("Loaded selfsigned certificate.")
	    return &certificate, err
	}
}

func InitWebServer2() {
	domain := "votezilla.io"

	certManager := autocert.Manager{
	  Prompt:     autocert.AcceptTOS,
	  HostPolicy: autocert.HostWhitelist(domain),
	  Cache:      autocert.DirCache("certs"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	    fmt.Fprint(w, "Hello HTTP/2")
	})

	tlsConfig := certManager.TLSConfig()
	tlsConfig.GetCertificate = getSelfSignedOrLetsEncryptCert(&certManager)
	
	server := http.Server{
	    Addr:    ":443",
	    Handler: mux,
	    TLSConfig: tlsConfig, 
	    	//&tls.Config{
	    	//  NextProtos: []string{"h2", "http/1.1"},
	    	//},
	}

	fmt.Printf("Server listening on %s", server.Addr)
	//go http.ListenAndServe(":80", mux)
	go http.ListenAndServe(":80", http.HandlerFunc(redirectHTTP))

	if err := server.ListenAndServeTLS("certs/localhost.crt", "certs/localhost.key"); err != nil {
	    fmt.Println(err)
	}
}*/
