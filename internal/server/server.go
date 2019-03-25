package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Server interface {
	Run() error
}

// server encapsulates all logic for registering and running a Server.
type server struct {
	Config Config
	Logger logrus.FieldLogger

	HTTPServer *http.Server
	GRPCServer *grpc.Server

	Shutdown func()

	// Exit chan for graceful Shutdown
	Exit chan chan error
}

func New(cfg Config, logger logrus.FieldLogger) *server {
	s := &server{
		Config: cfg,
		Logger: logger.WithField("component", "Server"),
		Exit:   make(chan chan error),
	}

	return s
}

func (s *server) start() error {
	go func() {
		err := s.HTTPServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.Logger.Errorf("HTTP Server error - initiating shutting down: %v", err)
			s.stop()
		}
	}()
	s.Logger.Infof("Listening and serving HTTP on %s", s.HTTPServer.Addr)

	if s.GRPCServer != nil {
		addr := fmt.Sprintf(":%d", s.Config.RPCPort)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			return errors.Wrap(err, "failed to listen to RPC port")
		}

		go func() {
			err := s.GRPCServer.Serve(lis)
			if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				err = nil
			}

			if err != nil {
				s.Logger.Errorf("gRPC Server error - initiating shutting down: %v", err)
				s.stop()
			}
		}()
		s.Logger.Infof("Listening on RPC port: %d", s.Config.RPCPort)
	}

	go func() {
		exit := <-s.Exit

		// stop listener with timeout
		ctx, cancel := context.WithTimeout(context.Background(), s.Config.ShutdownTimeout)
		defer cancel()

		// stop service
		if s.Shutdown != nil {
			s.Shutdown()
		}

		// stop gRPC Server
		if s.GRPCServer != nil {
			s.GRPCServer.GracefulStop()
		}

		// stop HTTP Server
		exit <- s.HTTPServer.Shutdown(ctx)
	}()

	return nil
}

func (s *server) stop() error {
	ch := make(chan error)
	s.Exit <- ch
	return <-ch
}

// Run will create a new Server and register the given
// Service and start up the Server(s).
// This will block until the Server shuts down.
func (s *server) Run() error {
	if err := s.start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	s.Logger.Info("Received signal ", <-ch)
	return s.stop()
}

func NewGRPCServer(service interface{}, desc *grpc.ServiceDesc, middleware grpc.UnaryServerInterceptor, options []grpc.ServerOption) *grpc.Server {
	var inters []grpc.UnaryServerInterceptor
	if mw := middleware; mw != nil {
		inters = append(inters, mw)
	}

	chain := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(inters...))
	interceptors := append(options, chain)

	server := grpc.NewServer(interceptors...)
	server.RegisterService(desc, service)

	return server
}

func NewHTTPServer(cfg Config, handler http.Handler) *http.Server {
	return &http.Server{
		Handler:        handler,
		Addr:           fmt.Sprintf(":%d", cfg.HTTPPort),
		MaxHeaderBytes: cfg.MaxHeaderBytes,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
	}
}
