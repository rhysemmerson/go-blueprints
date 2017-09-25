package main

import (
	"net/http"
	"strings"
	"fmt"
	"github.com/stretchr/objx"
	"github.com/stretchr/gomniauth"
)

type authHandler struct {
	next http.Handler
}
func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("auth")
	if err == http.ErrNoCookie {
		// not authenticated
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.next.ServeHTTP(w, r)
}
func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	segs := strings.Split(r.URL.Path, "/")
	var action string
	var provider string
	if len(segs) == 4 {
		action = segs[2]
		provider = segs[3]
	}
	switch action {
		case "login":
			provider, err := gomniauth.Provider(provider)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error when trying to get provider %s: %s", provider, err), http.StatusBadRequest)
				return
			}
			loginUrl, err := provider.GetBeginAuthURL(nil, nil)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error when trying to GetBeginAuthUrl for %s: %s", provider, err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Location", loginUrl)
			w.WriteHeader(http.StatusTemporaryRedirect)
			break
		case "callback": 
			provider, err := gomniauth.Provider(provider)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error when trying to get provider %s: %s", provider, err), http.StatusBadRequest)
				return
			}
			creds, err := provider.CompleteAuth(objx.MustFromURLQuery(r.URL.RawQuery))
			if err != nil {
				http.Error(w, fmt.Sprintf("Error when trying to complete auth for %s: %s", provider, err), http.StatusInternalServerError)
				return
			}
			user, err := provider.GetUser(creds)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error when trying to get user from %s: %s", provider, err), http.StatusInternalServerError)
				return
			}
			authCookieValue := objx.New(map[string]interface{}{
				"name": user.Name(),
				"avatar_url": user.AvatarURL(),
			}).MustBase64()
			http.SetCookie(w, &http.Cookie{
				Name: "auth",
				Value: authCookieValue,
				Path: "/",
			})
			w.Header().Set("Location", "/chat")
			w.WriteHeader(http.StatusTemporaryRedirect)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Auth action %s not supported", action)
	}
}