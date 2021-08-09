package main

import (
	"crypto/tls"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/mux"
)

func (p *Plugin) GetHTTPClient() *HTTPClient {
	config := p.getConfiguration()

	var client HTTPClient = HTTPClient{client: http.Client{}}

	if !config.DESEnableTLS {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = HTTPClient{client: http.Client{Transport: tr}}
	}

	return &client
}

func (p *Plugin) DebugRoutes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				p.API.LogError(
					"url", r.URL.String(),
					"error", x,
					"stack", string(debug.Stack()))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) forkRouter() *mux.Router {
	router := mux.NewRouter()
	router.Use(p.DebugRoutes)

	subrouter := router.PathPrefix("/onlyofficeapi").Subrouter()
	subrouter.HandleFunc("/download", p.downloadMiddleware(p.download)).Methods(http.MethodGet)
	subrouter.HandleFunc("/editor", p.userAccessMiddleware(p.editor)).Methods(http.MethodPost)
	subrouter.HandleFunc("/callback", p.callbackMiddleware(p.callback)).Methods(http.MethodPost)
	subrouter.HandleFunc("/set_file_permissions", p.permissionsMiddleware(p.setFilePermissions)).Methods(http.MethodPost)
	subrouter.HandleFunc("/get_file_permissions", p.permissionsMiddleware(p.getFilePermissions)).Methods(http.MethodGet)
	subrouter.HandleFunc("/channel_users", p.userAccessMiddleware(p.channelUsers)).Methods(http.MethodGet)
	subrouter.HandleFunc("/channel_user", p.userAccessMiddleware(p.userPermissions)).Methods(http.MethodGet)

	return router
}
