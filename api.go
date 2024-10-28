package blockchainlite

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

type Server struct {
	blockchain *Blockchain
	httpServer *http.Server
}
type Response struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func NewServer(bcName string) (*Server, error) {
	bc, err := NewBlockchain(bcName)
	if err != nil {
		return nil, err
	}
	return &Server{
		blockchain: bc,
		httpServer: &http.Server{},
	}, nil
}

func (s *Server) writeResponse(w http.ResponseWriter, code int, data interface{}, errMsg string) {
	response := Response{
		Code:  code,
		Data:  data,
		Error: errMsg,
	}
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) AddBlockHandler(w http.ResponseWriter, r *http.Request) {
	var data interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		s.writeResponse(w, http.StatusBadRequest, nil, err.Error())
		return
	}

	if err := s.blockchain.AddBlock(data); err != nil {
		s.writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	s.writeResponse(w, http.StatusCreated, "Block added successfully", "")
}

func (s *Server) GetLatestBlockHandler(w http.ResponseWriter, r *http.Request) {
	block, err := s.blockchain.GetLatestBlock()
	if err != nil {
		s.writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	if block == nil {
		s.writeResponse(w, http.StatusNotFound, nil, "No blocks found")
		return
	}
	s.writeResponse(w, http.StatusOK, block, "")
}

func (s *Server) GetBlockHistoryHandler(w http.ResponseWriter, r *http.Request) {
	blocks, err := s.blockchain.GetBlockHistory()
	if err != nil {
		s.writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}
	s.writeResponse(w, http.StatusOK, blocks, "")
}

func (s *Server) Start(addr string) error {
	r := mux.NewRouter()

	r.HandleFunc("/blocks", s.AddBlockHandler).Methods("POST")
	r.HandleFunc("/blocks/latest", s.GetLatestBlockHandler).Methods("GET")
	r.HandleFunc("/blocks/history", s.GetBlockHistoryHandler).Methods("GET")

	s.httpServer.Addr = addr
	log.Printf("Starting server on %s\n", addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	log.Println("Stopping server...")
	s.blockchain.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}
