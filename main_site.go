package main

// import (
// 	"fmt"
// 	"html/template"
// 	"net/http"
// 	"os"
// 	"path/filepath"
// 	"strconv"
// 	"sync"
// 	"time"
// )

// var viewCounter ViewCounter
// var tmpl *template.Template
// var once sync.Once
// var passwordActivated = false
// var appPassword = "nospoil"

// var cookieVal = &http.Cookie{
// 	Name:  "verified",
// 	Value: "==",
// 	Path:  "/",
// }

// type ViewCounter struct {
// 	ipCalled map[string]int
// 	count    int
// 	mutex    sync.Mutex
// 	filePath string
// }

// func (vc *ViewCounter) IncView(remoteAddr string) {
// 	if vc.ipCalled[remoteAddr] == 0 {
// 		vc.ipCalled[remoteAddr] = 1
// 	} else {
// 		vc.ipCalled[remoteAddr]++
// 		return
// 	}
// 	vc.mutex.Lock()
// 	vc.count++
// 	vc.mutex.Unlock()
// }

// func (vc *ViewCounter) loadViewCount() {
// 	data, err := os.ReadFile(vc.filePath)
// 	if err != nil && !os.IsNotExist(err) {
// 		os.Create(vc.filePath)
// 		return
// 	}

// 	if len(data) > 0 {
// 		vc.count, _ = strconv.Atoi(string(data))
// 	} else {
// 		vc.count = 0
// 	}
// }

// func (vc *ViewCounter) saveViewCount() {
// 	go func() {
// 		ticker := time.NewTicker(30 * time.Minute)
// 		for range ticker.C {
// 			err := os.WriteFile(vc.filePath, []byte(fmt.Sprintf("%d", vc.count)), 0644)
// 			if err != nil {
// 				fmt.Println("Error saving view count:", err)
// 			}
// 		}
// 	}()
// 	os.WriteFile(vc.filePath, []byte(fmt.Sprintf("%d", vc.count)), 0644)
// }

// func passwordVerification(password string) bool {
// 	return password == appPassword
// }

// func deactivateHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Query().Get("pwd") == "" {
// 		w.WriteHeader(http.StatusOK)
// 		passwordActivated = false
// 		http.Redirect(w, r, "/", http.StatusFound)
// 		return
// 	}
// }

// func activateHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Query().Get("pwd") == "" {
// 		w.WriteHeader(http.StatusOK)
// 		passwordActivated = true
// 		http.Redirect(w, r, "/", http.StatusFound)
// 		return
// 	}
// }

// func passwordFormHandler(w http.ResponseWriter, _ *http.Request) {
// 	tmpl.ExecuteTemplate(w, "password.html", nil)
// }

// func processpwdFormHandler(w http.ResponseWriter, r *http.Request) {
// 	password := r.FormValue("password")
// 	hiddenField := r.FormValue("hiddenField")
// 	if hiddenField == "" && passwordVerification(password) {
// 		//set cookie and redirect to index
// 		http.SetCookie(w, cookieVal)
// 		http.Redirect(w, r, "/", http.StatusFound)
// 	} else {
// 		fmt.Fprint(w, "You entered the wrong password!")
// 	}
// }

// func processContactFormHandler(w http.ResponseWriter, r *http.Request) {
// 	// email := r.FormValue("email")
// 	// subject := r.FormValue("subject")
// 	// message := r.FormValue("message")
// 	// accepted := r.FormValue("accept")
// 	// hiddenField := r.FormValue("happy")
// 	// if hiddenField == "" && accepted == "on" {
// 	// 	finalSubject := "Benedicte de choulot - " + subject
// 	// 	body := "<b>ATTENTION NE JAMAIS CLIQUER SUR UN LIEN (un lien facebook.com/abcd peut cacher phishing.fr/hack)</b>\n\nDe: <b>" + email + "</b>\n\n Message:\n" + message
// 	// 	finalMessage := []byte(finalSubject + body)

// 	// 	auth := smtp.PlainAuth("", fromGmail, gmailAppPwd, host)

// 	// 	err := smtp.SendMail(address, auth, fromGmail, toGmail, finalMessage)
// 	// 	if err != nil {
// 	// 		panic(err)
// 	// 	}
// 	// 	http.Redirect(w, r, "/contact?confirmation=yes", http.StatusMovedPermanently)
// 	// } else {
// 	// 	http.Redirect(w, r, "/contact?confirmation=no", http.StatusMovedPermanently)
// 	// }
// }

// func indexHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Path != "/" {
// 		notFoundHandler(w, r)
// 		return
// 	}
// 	//check if cookie is set and equal to cookieVal
// 	cook, err := r.Cookie("verified")
// 	if passwordActivated && err != nil || passwordActivated && cook.Value != cookieVal.Value {
// 		passwordFormHandler(w, r)
// 		return
// 	}
// 	viewCounter.IncView(r.RemoteAddr)
// 	tmpl.ExecuteTemplate(w, "index.html", nil)
// }

// func programHandler(w http.ResponseWriter, r *http.Request) {
// 	//check if cookie is set and equal to cookieVal
// 	cook, err := r.Cookie("verified")
// 	if passwordActivated && err != nil || passwordActivated && cook.Value != cookieVal.Value {
// 		passwordFormHandler(w, r)
// 		return
// 	}
// 	viewCounter.IncView(r.RemoteAddr)
// 	tmpl.ExecuteTemplate(w, "program.html", nil)
// }

// func campagneHandler(w http.ResponseWriter, r *http.Request) {
// 	//check if cookie is set and equal to cookieVal
// 	cook, err := r.Cookie("verified")
// 	if passwordActivated && err != nil || passwordActivated && cook.Value != cookieVal.Value {
// 		passwordFormHandler(w, r)
// 		return
// 	}
// 	viewCounter.IncView(r.RemoteAddr)
// 	tmpl.ExecuteTemplate(w, "campagne.html", nil)
// }

// func contactHandler(w http.ResponseWriter, r *http.Request) {
// 	//check if cookie is set and equal to cookieVal
// 	cook, err := r.Cookie("verified")
// 	if passwordActivated && err != nil || passwordActivated && cook.Value != cookieVal.Value {
// 		passwordFormHandler(w, r)
// 		return
// 	}
// 	viewCounter.IncView(r.RemoteAddr)
// 	data := struct {
// 		Confirmation string
// 	}{
// 		Confirmation: r.URL.Query().Get("confirmation"),
// 	}
// 	tmpl.ExecuteTemplate(w, "contact.html", data)
// }

// func notFoundHandler(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusNotFound)
// 	tmpl.ExecuteTemplate(w, "404.html", nil)
// }

// func viewCountHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Query().Get("pwd") == "BenedicteEthan2024" {
// 		w.WriteHeader(http.StatusOK)
// 		fmt.Fprintf(w, "View count: %d", viewCounter.count)
// 		return
// 	}
// 	w.WriteHeader(http.StatusUnauthorized)
// }

// func init() {
// 	once.Do(func() {
// 		filePath := filepath.Join(".", "view_count.txt")
// 		viewCounter = ViewCounter{ipCalled: make(map[string]int), count: 0, filePath: filePath}
// 		viewCounter.loadViewCount()
// 		tmpl = template.Must(template.ParseGlob("templates/*.html"))
// 		viewCounter.saveViewCount()
// 	})
// }

// func main() {
// 	// Set the HTTPS certificates and keys
// 	certFile := "./static/certs/cert.cer"
// 	keyFile := "./static/certs/cert.key"

// 	// Load the certificate and key into a TLS configuration
// 	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Create a new TLS configuration with the certificate and key
// 	tlsConfig := &tls.Config{
// 		Certificates: []tls.Certificate{cert},
// 	}

// 	// Create a new HTTPS server with the certificate and key
// 	httpsServer := &http.Server{
// 		Addr:      ":443",
// 		Handler:   http.DefaultServeMux,
// 		TLSConfig: tlsConfig,
// 	}

// 	// Define the handler functions for the main domain and "www" subdomain
// 	http.HandleFunc("/", indexHandler)
// 	http.HandleFunc("/index.html", indexHandler)
// 	http.HandleFunc("/programme", programHandler)
// 	http.HandleFunc("/campagne", campagneHandler)
// 	http.HandleFunc("/numberOfViews", viewCountHandler)
// 	http.HandleFunc("/process", processpwdFormHandler)
// 	http.HandleFunc("/contact", contactHandler)
// 	http.HandleFunc("/send", processContactFormHandler)
// 	http.HandleFunc("/deactivatepwd", deactivateHandler)
// 	http.HandleFunc("/activatepwd", activateHandler)

// 	// Serve static files
// 	imageDir := http.Dir("./static")
// 	robotFile := "robots.txt"
// 	sitemapFile := "sitemap.xml"
// 	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(imageDir)))
// 	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
// 		http.ServeFile(w, r, filepath.Join(robotFile))
// 	})
// 	http.HandleFunc("/sitemap", func(w http.ResponseWriter, r *http.Request) {
// 		http.ServeFile(w, r, filepath.Join(sitemapFile))
// 	})

// 	// Start the HTTP server to redirect www to non-www
// 	go http.ListenAndServe(":80", http.RedirectHandler("https://dechoulot2024.fr", http.StatusMovedPermanently))

// 	// Start the HTTPS server
// 	fmt.Printf("Starting HTTPS server on port 443 with certificates\n")
// 	err = httpsServer.ListenAndServeTLS("", "")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }
