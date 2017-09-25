package main

import (
	"log"
	"net/http"
	"flag"
	"sync"
	"text/template"
	"path/filepath"
	"os"
	"github.com/startDaemons/go-blueprints/trace"	// our trace package
	"github.com/stretchr/objx"
	"github.com/stretchr/gomniauth" 				// for oauth
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/joho/godotenv"						// for dotenv support
)

type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// Compiles and renders the template t.filename
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})

	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func main() {
	// load envs in dotenv file
	if err := godotenv.Load(); err != nil {
	  log.Fatal("Error loading .env file")
	}

	addr := os.Getenv("APP_ADDRESS")
	securityKey := os.Getenv("SECURITY_KEY")

	if securityKey == "" {
		log.Fatal("Set SECURITY_KEY to a random, unique security key")
	}
	
	// init flags
	faddr := flag.String("addr", "", "Address to listen for http requests on")
	flag.Parse()

	if *faddr != "" {
		addr = *faddr
	}

	if addr == "" {
		log.Fatal("Server address must be provided: use APP_ADDRESS env or --addr option")
	}

	// init oauth
	googleClientId := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if googleClientId == "" || googleClientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set")
	}

	gomniauth.SetSecurityKey(securityKey)
	gomniauth.WithProviders(
		// facebook.New(...)
		// github.New(...)
		google.New(googleClientId, googleClientSecret, "http://chat.blueprint.dev:9000/auth/callback/google"),
	)

	// create room
	r := newRoom()
	r.tracer = trace.New(os.Stdout) // log to stdout

	// register routes
	http.Handle("/room", r)
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "auth",
			Value: "",
			Path: "/",
			MaxAge: -1,
		})
		w.Header().Set("Location", "/chat")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	go r.run()

	log.Println("Starting web server on ", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}
