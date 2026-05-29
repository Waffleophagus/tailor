//go:build !dev

package server

import "net/http"

func (s *Server) registerDevRoutes(mux *http.ServeMux) {}
