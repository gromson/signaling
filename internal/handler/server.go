package handler

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	response "github.com/gromson/http-json-response"
	log "github.com/sirupsen/logrus"
)

const (
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
)

// Server serves web socket clients
type Server struct {
	upgrader  *websocket.Upgrader
	hub       *hub
	apiClient ApiClient
}

// NewServer returns a pointer to a newly created Server instance
func NewServer(upgrader *websocket.Upgrader, apiClient ApiClient) *Server {
	h := newHub()
	go h.run()

	return &Server{
		upgrader:  upgrader,
		hub:       h,
		apiClient: apiClient,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("could not upgrade a connection")
		response.NewInternalError().Respond(w)
		return
	}

	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.WithError(err).Error("error while setting read deadline")
	}

	conn.SetPongHandler(
		func(_ string) error {
			if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
				log.WithError(err).Error("error on trying to ping")
				return err
			}
			return nil
		},
	)

	go newClient(conn, s.hub, s.apiClient).run()
}
