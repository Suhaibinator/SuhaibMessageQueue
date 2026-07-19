package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	serverconfig "github.com/Suhaibinator/SuhaibMessageQueue/internal/config"
	"github.com/Suhaibinator/SuhaibMessageQueue/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.LookupEnv); err != nil {
		log.Printf("smq: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, lookup serverconfig.LookupEnv) error {
	cfg, err := serverconfig.Load(args, lookup)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	tlsConfig, err := cfg.TLS.LoadServerTLS()
	if err != nil {
		return fmt.Errorf("load TLS configuration: %w", err)
	}

	srv, err := server.NewServer(cfg.Port, cfg.DBPath, &server.ServerOptions{
		EnableMTLS:        cfg.TLS.Enabled,
		TLSConfig:         tlsConfig,
		MaxRecvMsgSize:    cfg.MaxRecvMsgSize,
		MaxSendMsgSize:    cfg.MaxSendMsgSize,
		AllowRemoteWrites: cfg.AllowRemoteWrites,
	})
	if err != nil {
		return err
	}
	return srv.Serve(ctx)
}
