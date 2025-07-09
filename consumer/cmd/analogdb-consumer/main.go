package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/evanofslack/analogdb-consumer/internal/clickhouse"
	"github.com/evanofslack/analogdb-consumer/internal/config"
	"github.com/evanofslack/analogdb-consumer/internal/kafka"
	"github.com/evanofslack/analogdb-consumer/internal/logging"
	"github.com/evanofslack/analogdb-consumer/internal/metrics"
	"github.com/evanofslack/analogdb-consumer/internal/process"
	"github.com/evanofslack/analogdb-consumer/internal/server"
)

const (
	defaultConfigPath = "config.yaml"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigChan; cancel() }()

	var cfgPath string
	flag.StringVar(&cfgPath, "config", defaultConfigPath, "path to config.yml")
	flag.Parse()

	cfg, err := config.New(cfgPath)
	if err != nil {
		err = fmt.Errorf("load config, err=%w", err)
		fatal(nil, err)
	}

	logger, err := logging.New(cfg.Log.Level, cfg.Log.Env, cfg.App.Name)
	if err != nil {
		err = fmt.Errorf("create logger, err=%w", err)
		fatal(nil, err)
	}
	logger.Info("Starting app",
		"app", cfg.App.Name,
		"version", cfg.App.Version,
		"env", cfg.App.Env,
	)

	metrics, err := metrics.New(logger.With("subsystem", "metrics"))
	if err != nil {
		fatal(logger, err)
	}

	ch, err := clickhouse.New(
		logger.With("subsystem", "clickhouse"),
		metrics,
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.Username,
		cfg.ClickHouse.Password,
		cfg.ClickHouse.Table,
		cfg.App.Name,
		cfg.App.Version,
		cfg.ClickHouse.MigrationEnabled,
		cfg.ClickHouse.MigrationPath,
	)
	defer ch.Close()
	if err != nil {
		err = fmt.Errorf("create clickhouse client, err=%w", err)
		fatal(logger, err)
	}
	if err := ch.Open(); err != nil {
		err = fmt.Errorf("open clickhouse connection, err=%w", err)
		fatal(logger, err)
	}

	batchTimeout, err := cfg.Kafka.BatchTimeout()
	if err != nil {
		logger.Error("Invalid batch timeout", "error", err)
		batchTimeout = time.Second * 10
	}

	consumer := kafka.New(
		logger.With("subsystem", "kafka"),
		metrics,
		cfg.Kafka.Brokers(),
		cfg.Kafka.Topic,
		cfg.Kafka.ConsumerGroup,
		cfg.Kafka.BatchSize,
		batchTimeout,
	)

	processor := process.New(logger.With("subsystem", "processor"), consumer, ch)
	defer processor.Stop()

	httpServer := server.New(logger.With("subsystem", "server"), cfg.Server.Port, cfg.App.Name, cfg.App.Version, cfg.App.Env)
	httpServer.AddHealthChecker("clickhouse", ch)
	httpServer.AddHealthChecker("kafka", consumer)

	var wg sync.WaitGroup

	// start HTTP server
	wg.Add(1)
	go func() {
		defer httpServer.Shutdown(ctx)
		defer wg.Done()
		err := httpServer.Start(ctx)
		if err != nil {
			err = fmt.Errorf("start http server, err=%w", err)
			fatal(logger, err)
		}
	}()

	// start processor
	wg.Add(1)
	go func() {
		defer processor.Stop()
		defer wg.Done()
		err := processor.Start(ctx)
		if err != nil && err != context.Canceled {
			err = fmt.Errorf("start processor, err=%w", err)
			fatal(logger, err)
		}
	}()

	// start metrics server
	if cfg.Metrics.Enabled {
		wg.Add(1)
		go func() {
			defer metrics.Close(ctx)
			defer wg.Done()
			err := metrics.Serve(ctx, cfg.Metrics.Port)
			if err != nil && err != context.Canceled {
				err = fmt.Errorf("start processor, err=%w", err)
				fatal(logger, err)
			}
		}()
	}

	// wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received")
	cancel()

	// wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// wait for shutdown
	select {
	case <-done:
		logger.Info("Clean shutdown completed")
	case <-time.After(30 * time.Second):
		err = fmt.Errorf("shutdown timer exceeded")
		fatal(logger, err)
	}
}

func fatal(logger *slog.Logger, err error) {
	if logger != nil {
		logger.Error("Fatal error, exiting", "error", err)
	} else {
		err := fmt.Errorf("fatal error, exiting; er=%w", err)
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
