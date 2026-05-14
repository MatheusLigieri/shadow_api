package server

import "net/http"

type Server struct {
	Mux *http.ServeMux
}

func New() *Server {
	return &Server{Mux: http.NewServeMux()}
}
