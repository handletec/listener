package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/handletec/listener"
	"github.com/handletec/listener/rest"
)

type ctxType int

const (
	ctxStr ctxType = iota + 1
	ctxInt
)

func main() {

	listenerTLS, err := listener.NewTLSConfigBuilder(true) // parameter indicates if we should use the OS CA certificate list, otherwise it creates an empty CA pool
	if nil != err {
		log.Println(err)
		os.Exit(1)
	}

	// certificate and private key can be rotated without restarting the application

	// set the certificate and private key for this app
	err = listenerTLS.SetCertKeyFile("/path/to/app.crt", "/path/to/app.key")
	if nil != err {
		log.Println(err)
		os.Exit(1)
	}

	restListener := rest.New()

	restHandler := rest.NewNewHandler()
	restHandler.Set(rest.MethodGet, "/server/list", serverList, serverListMiddleWare)
	restHandler.Set(rest.MethodGet, "/server/{type}/{id}", serverID, serverIDMiddleWare)

	grpUser := rest.NewGroup("/user", userGroupMiddleWare)
	grpUser.Set(rest.MethodGet, "/list", userList, userListMiddleWare)
	grpUser.Set(rest.MethodGet, "/{id}", userID, userIDMiddleWare)

	restRouter := rest.NewRouter("/api", routerMiddleWare)
	restRouter.AddGoup(grpUser)
	restRouter.SetHandler(restHandler) // set all handlers under the base

	restConfig := rest.NewConfig()
	restConfig.EnableCompress(true) // enable output compression, useful for large amounts of data
	restConfig.SetRouter(restRouter)

	err = restListener.SetConfig(restConfig)
	if nil != err {
		log.Println(err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	//_ = logger

	restListener.Init(logger, rest.DefaultAddr, rest.DefaultPort, listenerTLS.ForServer())
	err = restListener.Start()
	if nil != err {
		log.Println(err)
		os.Exit(1)
	}
}
func serverList(w http.ResponseWriter, r *http.Request) {
	log.Println("servers list called")
}

func serverID(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	id := ctx.Value(ctxInt)
	srvType := ctx.Value(ctxStr)
	log.Println("server id called with id", id, "for server type", srvType)
}

// middlewares must have the following structure
func serverListMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		log.Println("calling server list middleware")

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

// middlewares must have the following structure
func serverIDMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		id := chi.URLParam(r, "id")
		srvType := chi.URLParam(r, "type")

		ctx = context.WithValue(ctx, ctxStr, srvType)
		ctx = context.WithValue(ctx, ctxInt, id)

		log.Println("calling server id middleware with id", id, "for server type", srvType)

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func userList(w http.ResponseWriter, r *http.Request) {
	log.Println("user list called")
}

func userID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := ctx.Value(ctxInt)
	log.Println("user id called with id", id)
}

// middlewares must have the following structure
func userGroupMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		log.Println("calling user group middleware")

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

// middlewares must have the following structure
func userListMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		log.Println("calling user list middleware")

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

// middlewares must have the following structure
func userIDMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		id := chi.URLParam(r, "id")

		ctx = context.WithValue(ctx, ctxInt, id)

		log.Println("calling user id middleware with id", id)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

// middlewares must have the following structure
func routerMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		log.Println("calling router middleware")

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
