package server

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/derbylock/async-integration-testing/cmd/server/asit_api"
	"github.com/derbylock/async-integration-testing/cmd/server/cors"
	"github.com/derbylock/async-integration-testing/cmd/server/debug_api"
	"github.com/derbylock/async-integration-testing/cmd/server/health"
	"github.com/derbylock/async-integration-testing/cmd/server/requestlogger"
	"github.com/derbylock/async-integration-testing/internal/db"
	"github.com/julienschmidt/httprouter"
)

type Server struct {
	clientsRepository db.ClientsRepository
	port              int
}

func NewServer(storage db.Storage) *Server {
	clientsRepository := db.NewKVClientsRepository(storage)

	return &Server{
		clientsRepository: clientsRepository,
		port:              9580,
	}
}

const asitAPIPrefix = "/asit/api/v1"

func (s *Server) ListenAndServe() error {
	log.Println("Starting HTTP server")

	router := httprouter.New()
	health.InitAPIRoutes(asitAPIPrefix, router)
	asit_api.NewClientsAPIController(s.clientsRepository).InitRoutes(asitAPIPrefix, router)
	debug_api.InitAPIRoutes(asitAPIPrefix, router)

	log.Printf("Listening on port %d \r\n", *&s.port)
	handler := cors.NewCorsRouter(router)
	handlerLogger := requestlogger.Logger(os.Stdout, handler)
	handlerWithGZip := gziphandler.GzipHandler(handlerLogger)

	httpServer := &http.Server{
		Addr:           ":" + strconv.Itoa(s.port),
		Handler:        handlerWithGZip,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    300 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	httpServer.SetKeepAlivesEnabled(true)
	return httpServer.ListenAndServe()
}
