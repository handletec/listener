/*
Copyright © 2024 Vicknesh Suppramaniam <vicknesh@handletec.my>

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
package rest

import (
	"compress/flate"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	slogchi "github.com/samber/slog-chi"
	slogformatter "github.com/samber/slog-formatter"
)

const (
	// DefaultAddr - listen on all IPv4 and IPv6 interfaces
	DefaultAddr = "[::]"

	// DefaultPort - default port to listen on
	DefaultPort = 8081
)

// Listener - implementation of REST listener
type Listener struct {
	address   string
	port      int
	tlsConfig *tls.Config
	logger    *slog.Logger
	config    *Config
	header    *Header
}

// New - create new instance of the REST listener
func New() (l *Listener) {
	l = new(Listener)
	return
}

// Name - returns the name of this listener
func (l *Listener) Name() (str string) {
	return "REST"
}

// Init - initializes this listener with any necessary configuration parameters
func (l *Listener) Init(logger *slog.Logger, address string, port int, tlsConfig *tls.Config) (err error) {

	if len(address) == 0 {
		address = DefaultAddr // if no address is given we have it listen on all IPv4 and IPv6 interfaces
	}

	if port == 0 {
		port = DefaultPort // default port if none is given
	}

	l.address = address
	l.port = port
	l.tlsConfig = tlsConfig

	if nil == logger {
		// if no logger is given, create a new instance
		logger = slog.New(
			slogformatter.NewFormatterHandler(
				slogformatter.TimezoneConverter(time.UTC),
				slogformatter.TimeFormatter(time.RFC3339, nil),
			)(
				//slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}),
				slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}),
			),
		)

		// Add an attribute to all log entries made through this logger.
		//logger = logger.With("env", "production")
	}

	l.logger = logger

	if nil == l.config {
		// if no configuration is set, init a new one with sane default values
		l.config = NewConfig()
	}

	return
}

// SetConfig - sets configuration details for this lietener
func (l *Listener) SetConfig(config any) (err error) {
	l.config = config.(*Config)
	return
}

// Start - starts this listener
func (l *Listener) Start() (err error) {
	l.logger.Info("listener starting", "listener", l.Name())

	if nil == l.config.router {
		return errors.New("REST start: no HTTP routers configured")
	}

	router := chi.NewRouter()

	router.Use(slogchi.New(l.logger.WithGroup(l.Name())))

	/*
		// print the requests information
		if l.config.Log {
			//router.Use(middleware.Logger)
			//router.Use(slogchi.New(l.logger)) // this throws an panic, must troubleshoot

			router.Use(slogchi.New(logger.WithGroup("rest")))
		}
	*/

	if l.config.compress {
		router.Use(middleware.Compress(flate.DefaultCompression)) // compress data for smaller size
	}

	router.Use(middleware.RealIP)

	// (optional) - do not cache requests
	router.Use(middleware.NoCache)

	router.Use(middleware.Throttle(l.config.RPS)) // restrict number of concurrent requests per second

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	router.Use(middleware.Timeout(l.config.Timeout))

	router.Use(render.SetContentType(render.ContentTypeJSON))
	router.Use(middleware.AllowContentType("application/json")) // only accept JSON content type
	router.Use(middleware.Recoverer)

	// CORS configuration
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   l.config.CORS.AllowedOrigins,
		AllowedMethods:   l.config.CORS.AllowedMethods,
		AllowedHeaders:   l.config.CORS.AllowedHeaders,
		AllowCredentials: l.config.CORS.AllowCredentials,
		ExposedHeaders:   l.config.CORS.AllowedHeaders,
		MaxAge:           l.config.CORS.MaxAge, // Maximum value not ignored by any of major browsers
		Debug:            l.config.CORS.Debug,
	}))

	router.Use(headerMiddleware(l.header))

	// handle OPTIONS request, usually for CORS, though the CORS handler above does the heavy lifting for us already
	l.config.router.r.MethodFunc(MethodOptions.String(), PatternAll, optionsHandler(l.config.CORS))

	//router.Mount("/", l.config.router.r) // mount the root to the given handler

	l.config.router.mount()              // mount all the paths
	router.Mount("/", l.config.router.r) // mount the root to the given handler

	/*
		for _, route := range l.config.router.r.Routes() {
			fmt.Println(route.Pattern, route.SubRoutes)
		}
	*/

	address := fmt.Sprintf("%s:%d", l.address, l.port)

	if nil != l.tlsConfig {
		l.logger.Info("listener started", "listener", l.Name(), "address", "https://"+address, "tls", "true")

		// start HTTPS server
		server := &http.Server{
			Addr:      address,
			Handler:   router,
			TLSConfig: l.tlsConfig,
		}

		err = server.ListenAndServeTLS("", "")
		if nil != err {
			return fmt.Errorf("start rest: %w", err)
		}

	} else {
		l.logger.Info("listener started", "listener", l.Name(), "address", "http://"+address, "tls", "false")

		// start normal HTTP server
		err = http.ListenAndServe(address, router)

	}

	return
}
