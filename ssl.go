// HTTPS SSL.  Sample ref: https://marcofranssen.nl/build-a-go-webserver-on-http-2-using-letsencrypt/
//
//  TODO: see also: https://gist.github.com/samthor/5ff8cfac1f80b03dfe5a9be62b29d7f2
//                  https://goenning.net/2017/11/08/free-and-automated-ssl-certificates-with-go/

package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"path/filepath"
	"golang.org/x/crypto/acme/autocert"
	"time"
)

var (
	domain string
)

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

func InitWebServer() {
	mux := SetupWebHandlers()

	server := http.Server {
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	domains := []string{"votezilla.io", "www.votezilla.io", "votezilla.news", "www.votezilla.news"}

	if flags.domain != "" {
		// Running SSL in production - votezilla.io
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domains...),
			Cache:      autocert.DirCache("certs"),
		}

		tlsConfig := certManager.TLSConfig()
		tlsConfig.GetCertificate = getSelfSignedOrLetsEncryptCert(&certManager)

		server.Addr		 = ":443"
		server.TLSConfig = tlsConfig

		go http.ListenAndServe(":80", certManager.HTTPHandler(nil))

		fmt.Println("Server listening on", server.Addr)
		check(server.ListenAndServeTLS("", ""))
	} else {
		// Running on localhost
		server.Addr		= ":" + flags.port

		fmt.Println("Server listening on", server.Addr)
		check(server.ListenAndServe())
	}
}
