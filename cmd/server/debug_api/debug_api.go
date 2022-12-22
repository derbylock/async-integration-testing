package debug_api

import (
	"net/http"
	"net/http/pprof"

	"github.com/julienschmidt/httprouter"
)

func InitAPIRoutes(pathPrefix string, router *httprouter.Router) {
	router.HandlerFunc(http.MethodGet, pathPrefix+"/debug/pprof/", pprof.Index)
	router.HandlerFunc(http.MethodGet, pathPrefix+"/debug/pprof/cmdline", pprof.Cmdline)
	router.HandlerFunc(http.MethodGet, pathPrefix+"/debug/pprof/profile", pprof.Profile)
	router.HandlerFunc(http.MethodGet, pathPrefix+"/debug/pprof/symbol", pprof.Symbol)
	router.HandlerFunc(http.MethodGet, pathPrefix+"/debug/pprof/trace", pprof.Trace)
	router.Handler(http.MethodGet, pathPrefix+"/debug/pprof/allocs", pprof.Handler("allocs"))
	router.Handler(http.MethodGet, pathPrefix+"/debug/pprof/block", pprof.Handler("block"))
	router.Handler(http.MethodGet, pathPrefix+"debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handler(http.MethodGet, pathPrefix+"/debug/pprof/heap", pprof.Handler("heap"))
	router.Handler(http.MethodGet, pathPrefix+"/debug/pprof/mutex", pprof.Handler("mutex"))
	router.Handler(http.MethodGet, pathPrefix+"/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
}
