package rest

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strings"
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

	// WS: used by WebSocket integration
	wsHandler WSHandler
	wsHub     *WSHub
	server    *http.Server
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
				slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}),
			),
		)
	}
	l.logger = logger

	if nil == l.config {
		// if no configuration is set, init a new one with sane default values
		l.config = NewConfig()
	}

	return
}

// SetConfig - sets configuration details for this listener
func (l *Listener) SetConfig(config any) (err error) {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("setconfig: want '*rest.Config', got %T", config)
	}
	l.config = cfg
	return nil
}

// SetWSHandler - application-level per-connection WebSocket handler
func (l *Listener) SetWSHandler(h WSHandler) { l.wsHandler = h }

// Start - starts this listener
func (l *Listener) Start() (err error) {
	l.logger.Info("listener starting", "listener", l.Name())

	if nil == l.config.router {
		return errors.New("REST start: no HTTP routers configured")
	}

	router := chi.NewRouter()

	// Common middlewares for everything (safe for WS and REST):
	router.Use(slogchi.New(l.logger.WithGroup(l.Name())))
	router.Use(middleware.RealIP)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					l.logger.Error("request panic", "err", rec)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	})
	router.Use(middleware.NoCache)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	})
	router.Use(limitRequestBody(l.config.maxBody)) // set max limit for the body

	// --- WebSocket route: NO JSON content-type, NO global Timeout ---
	if l.config.WS != nil && l.config.WS.Enabled {
		l.logger.Info("websocket enabled", "listener", l.Name(), "path", l.config.WS.Path)

		// init hub once
		l.wsHub = NewWSHub(l.logger, l.config.WS)
		go l.wsHub.Run()

		router.Group(func(r chi.Router) {
			// Optional: throttle the handshake
			if l.config.MaxConcurrent > 0 {
				r.Use(middleware.Throttle(l.config.MaxConcurrent))
			}
			r.Get(l.config.WS.Path, l.wsAccept)
		})
	}

	// --- REST subrouter: keep existing REST middlewares ---
	api := chi.NewRouter()

	if l.config.compress {
		//api.Use(middleware.Compress(flate.DefaultCompression)) // compress data for smaller size
		api.Use(middleware.Compress(gzip.DefaultCompression)) // or: middleware.Compress(-1)
	}

	api.Use(middleware.Throttle(l.config.MaxConcurrent)) // restrict number of concurrent requests per second
	api.Use(middleware.Timeout(l.config.Timeout))        // REST-only timeout

	api.Use(render.SetContentType(render.ContentTypeJSON))
	//api.Use(middleware.AllowContentType("application/json")) // only accept JSON content type

	api.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ct := r.Header.Get("Content-Type")
			if ct == "" {
				next.ServeHTTP(w, r)
				return
			}
			mediatype, _, _ := mime.ParseMediaType(ct)
			if mediatype != "application/json" && !strings.HasSuffix(mediatype, "+json") {
				http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// CORS configuration
	api.Use(cors.Handler(cors.Options{
		AllowedOrigins:   l.config.CORS.AllowedOrigins,
		AllowedMethods:   l.config.CORS.AllowedMethods,
		AllowedHeaders:   l.config.CORS.AllowedHeaders,
		AllowCredentials: l.config.CORS.AllowCredentials,
		MaxAge:           l.config.CORS.MaxAge, // Maximum value not ignored by major browsers
		Debug:            l.config.CORS.Debug,
		//ExposedHeaders:   l.config.CORS.AllowedHeaders,
	}))

	l.logger.Debug("CORS", "allowed origins", l.config.CORS.AllowedOrigins, "allowed methods", l.config.CORS.AllowedMethods, "allowed headers", l.config.CORS.AllowedHeaders)

	api.Use(headerMiddleware(l.header))

	// handle OPTIONS request (REST only)
	//l.config.router.r.MethodFunc(MethodOptions.String(), PatternAll, optionsHandler(l.config.CORS))

	// mount application routes
	l.config.router.mount()
	api.Mount("/", l.config.router.r)
	router.Mount("/", api)

	address := fmt.Sprintf("%s:%d", l.address, l.port)

	// Build server so we can shutdown gracefully later and to ensure 'server' is used.
	l.server = &http.Server{
		Addr:              address,
		Handler:           router,
		TLSConfig:         l.tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if nil != l.tlsConfig && len(l.tlsConfig.Certificates) > 0 {
		l.logger.Info("listener started", "listener", l.Name(), "address", "https://"+address, "tls", "true")
		err = l.server.ListenAndServeTLS("", "")
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("start rest: %w", err)
		}
	} else {
		l.logger.Info("listener started", "listener", l.Name(), "address", "http://"+address, "tls", "false")
		err = l.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("start rest: %w", err)
		}
	}

	return
}

// Optional graceful stop
func (l *Listener) Stop(ctx context.Context) error {
	if l.wsHub != nil {
		l.wsHub.Close()
	}
	if l.server != nil {
		return l.server.Shutdown(ctx)
	}
	return nil
}
