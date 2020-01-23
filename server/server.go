package server

/*
   Copyright 2019 Bruno Moura <brunotm@gmail.com>

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"net/http"
	"time"

	"github.com/brunotm/replicant/log"
	"github.com/brunotm/replicant/manager"
	"github.com/julienschmidt/httprouter"
)

// Config for replicant server
type Config struct {
	Username          string        `json:"username" yaml:"username"`
	Password          string        `json:"password" yaml:"password"`
	ListenAddress     string        `json:"listen_address" yaml:"listen_address"`
	WriteTimeout      time.Duration `json:"write_timeout" yaml:"write_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout" yaml:"read_timeout"`
	ReadHeaderTimeout time.Duration `json:"read_header_timeout" yaml:"read_header_timeout"`
}

// Server is an replicant manager and api server
type Server struct {
	config  Config
	http    *http.Server
	router  *httprouter.Router
	manager *manager.Manager
}

// New creates a new replicant server
func New(config Config, m *manager.Manager, r *httprouter.Router) (server *Server, err error) {
	server = &Server{}
	server.manager = m
	server.config = config
	server.router = r
	server.http = &http.Server{}
	server.http.Addr = config.ListenAddress

	if config.WriteTimeout != 0 {
		server.http.WriteTimeout = config.WriteTimeout
	}

	if config.ReadTimeout != 0 {
		server.http.ReadTimeout = config.ReadTimeout
	}

	if config.ReadHeaderTimeout != 0 {
		server.http.ReadHeaderTimeout = config.ReadHeaderTimeout
	}

	server.http.Handler = server.router
	return server, nil
}

// Start serving
func (s *Server) Start() (err error) {
	if err = s.http.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server: error starting http: %w", err)
	}
	return nil
}

// Router returns this server http router
func (s *Server) Router() (r *httprouter.Router) {
	return s.router
}

// Manager returns this server replicant manager
func (s *Server) Manager() (m *manager.Manager) {
	return s.manager
}

// ServeHTTP implements the http.Handler interface for testing and handler usage
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

// Close this server
func (s *Server) Close(ctx context.Context) (err error) {
	s.http.Shutdown(ctx)
	return s.manager.Close()
}

// AddHandler adds a handler for the given method and path
func (s *Server) AddHandler(method, path string, handler Handler) {
	log.Info("adding handler").String("path", path).String("method", method).Log()

	if s.config.Username != "" && s.config.Password != "" {
		handler = basicAuth(handler, s.config.Username, s.config.Password)
	}

	s.router.Handle(method, path, logger(recovery(handler)))
}

// AddServerHandler adds a handler for the given method and path
func (s *Server) AddServerHandler(method, path string, handler ServerHandler) {
	log.Info("adding handler").String("path", path).String("method", method).Log()

	var h Handler
	switch s.config.Username != "" && s.config.Password != "" {
	case true:
		h = basicAuth(handler(s), s.config.Username, s.config.Password)
	case false:
		h = handler(s)
	}

	s.router.Handle(method, path, logger(recovery(h)))
}

// ServerHandler is handler that has access to the server
type ServerHandler func(*Server) httprouter.Handle

// Handler is a http handler
type Handler = httprouter.Handle

// Params from the URL
type Params = httprouter.Params

// recovery middleware
func recovery(h Handler) (n Handler) {
	return func(w http.ResponseWriter, r *http.Request, p Params) {

		defer func() {
			err := recover()
			if err != nil {
				log.Error("recovered from panic").
					String("error", fmt.Sprintf("%s", err)).
					String("stack", string(debug.Stack())).Log()

				jsonBody, _ := json.Marshal(map[string]interface{}{
					"message": "There was an internal server error",
					"error":   err,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			}
		}()

		h(w, r, p)
	}
}

// logger middleware
func logger(h Handler) (n Handler) {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		h(sw, r, p)
		log.Info("api request").String("method", r.Method).
			String("uri", r.URL.String()).
			String("requester", r.RemoteAddr).
			Int("status", int64(sw.status)).
			Int("content_length", int64(sw.length)).
			Int("duration_ms", time.Since(start).Milliseconds()).
			Log()
	}
}

// basic auth middleware until we have proper auth
func basicAuth(h Handler, user, password string) (n Handler) {
	return func(w http.ResponseWriter, r *http.Request, ps Params) {
		user, password, hasAuth := r.BasicAuth()
		if hasAuth && user == user && password == password {
			h(w, r, ps)
		} else {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	length int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	w.length += n
	return n, err
}
