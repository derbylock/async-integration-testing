package server

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/derbylock/async-integration-testing/cmd/server/cors"
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

func (s *Server) ListenAndServe() error {
	log.Println("Starting HTTP server")

	router := httprouter.New()
	router.GET("/asit/v1/health", health.GetHealthRoute)

	// router.POST("/repo/files/*filename", uploadFilesMultipart)
	// router.PUT("/repo/files/*filename", uploadFile)
	// router.DELETE("/repo/files/*filename", removeFile)
	// router.GET("/repo/files/*filename", downloadFile)

	// router.GET("/git/exists", gitExists)
	// router.POST("/git/clone", gitClone)
	// router.POST("/git/commit", gitCommit)
	// router.POST("/git/push", gitPush)
	// router.POST("/git/pull", gitPull)

	router.HandlerFunc(http.MethodGet, "/asit/v1/debug/pprof/", pprof.Index)
	router.HandlerFunc(http.MethodGet, "/asit/v1/debug/pprof/cmdline", pprof.Cmdline)
	router.HandlerFunc(http.MethodGet, "/asit/v1/debug/pprof/profile", pprof.Profile)
	router.HandlerFunc(http.MethodGet, "/asit/v1/debug/pprof/symbol", pprof.Symbol)
	router.HandlerFunc(http.MethodGet, "/asit/v1/debug/pprof/trace", pprof.Trace)
	router.Handler(http.MethodGet, "/asit/v1/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handler(http.MethodGet, "/asit/v1/debug/pprof/heap", pprof.Handler("heap"))
	router.Handler(http.MethodGet, "/asit/v1/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handler(http.MethodGet, "/asit/v1/debug/pprof/block", pprof.Handler("block"))

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
