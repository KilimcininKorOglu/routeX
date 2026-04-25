package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	v1 "routex/api/v1"
	"routex/app"
	"routex/constant"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

func SetupUnixSocket(a app.Main, errChan chan error) (*http.Server, error) {
	if err := os.Remove(constant.SockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to remove existing unix socket: %w", err)
	}

	socket, err := net.Listen("unix", constant.SockPath)
	if err != nil {
		return nil, fmt.Errorf("unix socket listen error: %v", err)
	}
	if err := os.Chmod(constant.SockPath, 0600); err != nil {
		socket.Close()
		return nil, fmt.Errorf("failed to set unix socket permissions: %w", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Mount("/api/v1", v1.NewRouter(a))

	srv := &http.Server{
		Handler: r,
	}

	log.Info().Msgf("Starting UNIX socket on %s", constant.SockPath)
	go func() {
		if e := srv.Serve(socket); e != nil && e != http.ErrServerClosed {
			errChan <- fmt.Errorf("unix socket server error: %v", e)
		}
		socket.Close()
		os.Remove(constant.SockPath)
	}()

	return srv, nil
}
