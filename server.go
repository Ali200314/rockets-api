package rocketsapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"rockets-api/rocket"
	"strings"
	"time"
)

type metadata struct {
	Channel       string    `json:"channel"`
	MessageType   string    `json:"messageType"`
	MessageNumber int       `json:"messageNumber"`
	MessageTime   time.Time `json:"messageTime"`
}

type server struct {
	store RocketStore
}

func NewServer(store RocketStore) *server {
	return &server{store: store}
}

func (s *server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /messages", s.handleRocketMessage)
	mux.HandleFunc("GET /rockets/{channel}", s.getRocket)
	mux.HandleFunc("GET /rockets", s.getRockets)
	return mux
}

func (s *server) handleRocketMessage(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Metadata metadata        `json:"metadata"`
		Message  json.RawMessage `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := payload.Metadata.validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	decodedMsg, err := rocket.DecodeMessage(payload.Metadata.MessageType, payload.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.processRocketMessage(
		payload.Metadata.Channel,
		payload.Metadata.MessageNumber,
		payload.Metadata.MessageTime,
		decodedMsg,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) getRocket(w http.ResponseWriter, r *http.Request) {
	channel := r.PathValue("channel")
	if channel == "" {
		http.Error(w, "missing rocket channel", http.StatusBadRequest)
		return
	}

	rocket, ok := s.store.getRocket(channel)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, rocket)
}

func (s *server) getRockets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.getRockets())
}

func (m metadata) validate() error {
	if strings.TrimSpace(m.Channel) == "" {
		return errors.New("channel is required")
	}

	if strings.TrimSpace(m.MessageType) == "" {
		return errors.New("messageType is required")
	}

	if m.MessageNumber <= 0 {
		return errors.New("messageNumber must be greater than zero")
	}

	if m.MessageTime.IsZero() {
		return errors.New("messageTime is required")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
