package main

// https://blog.kowalczyk.info/article/Jl3G/https-for-free-in-go.html
// To run:
// go run main.go
// Command-line options:
//   -production : enables HTTPS on port 443
//   -redirect-to-https : redirect HTTP to HTTTPS

import (
	"context"
	"crypto/tls"
	//"flag"
	"fmt"
	//"io"
	"log"
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

func InitWebServer2() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	    fmt.Fprint(w, "Hello HTTP/2")
	})

	server := http.Server{
	    Addr:    ":443",
	    Handler: mux,
	    TLSConfig: &tls.Config{
	      NextProtos: []string{"h2", "http/1.1"},
	    },
	}

	fmt.Printf("Server listening on %s", server.Addr)
	if err := server.ListenAndServeTLS("certs/localhost.crt", "certs/localhost.key"); err != nil {
	    fmt.Println(err)
	}
}
