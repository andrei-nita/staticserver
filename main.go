package main

import (
	_ "embed"
	"flag"
	"fmt"
	"github.com/andrei-nita/staticserver/middleware"
	"github.com/fatih/color"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var (
	portHTTP = ":8080"
	portSSL  = ":8443"
	ssl      bool
	gzip     bool
	cache    bool
	blue     *color.Color
	green    *color.Color
)

//go:embed mkcert/mkcert-windows-amd64.exe
var mkcert []byte

func init() {
	blue = color.New(color.FgBlue, color.Bold)
	green = color.New(color.FgGreen, color.Bold)
}

func main() {
	// create mkcertFile and add []byte to it
	mkcertFile, err := os.Create("mkcert.exe")
	if err != nil {
		log.Fatalln(err)
	}

	// write []byte to mkcertFile
	_, err = mkcertFile.Write(mkcert)
	if err != nil {
		log.Fatalln(err)
	}

	mkcertFile.Close()

	serverHTTP := http.Server{
		Addr: portHTTP,
	}

	serverSSL := http.Server{
		Addr: portSSL,
	}

	sslPtr := flag.Bool("ssl", false, "Activate HTTPS server")
	gzipPtr := flag.Bool("gzip", false, "Activate gzip")
	cachePtr := flag.Bool("cache", false, "Activate cache")
	defaultPtr := flag.Bool("default", false, "Use defaults, with everything deactivated")
	flag.Parse()

	if *defaultPtr {
		blue.Println("Every flag (-ssl, -gzip, -cache) is set to false")
	} else if *sslPtr || *gzipPtr || *cachePtr {
		ssl = *sslPtr
		gzip = *gzipPtr
		cache = *cachePtr
	} else {
		// questions
		ssl = doYouWant("ssl")
		gzip = doYouWant("gzip")
		cache = doYouWant("cache")
	}

	green.Printf("\nFlags: -ssl=%t -gzip=%t -cache=%t\n", ssl, gzip, cache)

	// gzip logic
	if gzip {
		serverSSL.Handler = new(middleware.GzipMiddleware)
		serverHTTP.Handler = new(middleware.GzipMiddleware)
	}

	// cache logic
	if cache {
		http.Handle("/", middleware.Cache(http.FileServer(http.Dir("."))))
	} else {
		http.Handle("/", http.FileServer(http.Dir(".")))
	}

	// ssl logic
	if ssl {
		// mkcert install CA
		blue.Println("Checking if CA is in the system trust store")
		err = exec.Command(mkcertFile.Name(), "-install").Run()
		if err != nil {
			log.Fatalln(err)
		}
		green.Println("CA is installed in the system trust store.")

		// mkcert create ssl certificates
		blue.Println("Creating SSL certificates")
		err = exec.Command(mkcertFile.Name(), "localhost", "127.0.0.1", "::1").Run()
		if err != nil {
			log.Fatalln(err)
		}
		green.Println("SSL certificates created.")

		serverHTTP.Handler = http.HandlerFunc(redirectTLS)

		fmt.Println("Navigate to: " + blue.Sprintf("https://localhost%s", portSSL))
		go func() {
			log.Fatalln(serverSSL.ListenAndServeTLS("localhost+2.pem", "localhost+2-key.pem"))
		}()
	} else {
		fmt.Println("Navigate to: " + blue.Sprintf("http://localhost%s", portHTTP))

	}

	log.Fatalln(serverHTTP.ListenAndServe())
}

func doYouWant(option string) (yes bool) {
	answer := ""

	// ask question
	blue.Printf("Activate %s? (y/N) ", option)

	// get answer
	_, err := fmt.Fscanln(os.Stdin, &answer)
	if err != nil {
		if strings.Contains(err.Error(), "unexpected newline") {
			//	ignore error
		} else {
			log.Fatalln(err)
		}
	}

	if answer != "" {
		if strings.ToLower(answer) == "y" {
			yes = true
		}
	}

	return
}

func redirectTLS(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://localhost%s%s", portSSL, r.RequestURI)
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}
