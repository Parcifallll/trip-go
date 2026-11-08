package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Parcifallll/trip-go/internal/domain"
	"github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	HTTPAddr              string
	ShutdownTimeout       time.Duration
	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
}

type Server struct {
	httpServer *http.Server
	cfg        Config
}

func NewServer(
	cfg Config,
	tripHandler *handler.TripHandler,
	healthHandler *handler.HealthHandler,
) *Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	apiHandler := api.HandlerWithOptions(
		&apiServer{
			tripHandler:   tripHandler,
			healthHandler: healthHandler,
		},
		api.ChiServerOptions{
			BaseRouter: r,
			ErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
				log.Printf("request params error: %s %s: %v", req.Method, req.URL.Path, err)
				handler.WriteError(w, domain.ErrInvalidRequest, req.URL.Path)
			},
		},
	)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apiHandler,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		cfg:        cfg,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Run() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server error: %v", err)
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	return s.Shutdown(ctx)
}

type apiServer struct {
	tripHandler   *handler.TripHandler
	healthHandler *handler.HealthHandler
}

func (s *apiServer) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.tripHandler.CreateTrip(w, r)
}

func (s *apiServer) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.tripHandler.GetTrip(w, r, tripId)
}

func (s *apiServer) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.tripHandler.FinishTrip(w, r, tripId)
}

func (s *apiServer) Health(w http.ResponseWriter, r *http.Request) {
	s.healthHandler.Health(w)
}

func (s *apiServer) Ready(w http.ResponseWriter, r *http.Request) {
	s.healthHandler.Ready(w, r)
}

// ListTripPositions и CreateTripPosition будут реализованы в следующих ЛР
func (s *apiServer) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *apiServer) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}
