// Command myturn runs a small local-network web server that shows a group
// of people whose turn it is for a daily rotating responsibility.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myturn/internal/config"
	"myturn/internal/httpserver"
)

//go:embed web
var webFS embed.FS

const defaultConfigPath = "data/config.json"
const defaultAdminSecret = "changeme"
const defaultPort = "8031"

// version is set at build time via -ldflags "-X main.version=..."; see VERSION
// and the Dockerfile's VERSION build arg.
var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath := os.Getenv("MYTURN_CONFIG")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("preparing static assets: %w", err)
	}

	adminSecret := os.Getenv("MYTURN_ADMIN_SECRET")
	if adminSecret == "" {
		adminSecret = defaultAdminSecret
		log.Printf("warning: MYTURN_ADMIN_SECRET not set, using default admin secret %q (set it before exposing this beyond a trusted LAN)", defaultAdminSecret)
	}

	srv := httpserver.New(cfg, configPath, staticFS, adminSecret)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	addr := "0.0.0.0:" + port

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	printStartupInfo(port)

	errCh := make(chan error, 1)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Println("shutting down...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	log.Println("stopped")
	return nil
}

// loadOrCreateConfig loads the config file, creating a sensible default one
// if it does not already exist.
func loadOrCreateConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		defaultCfg := &config.Config{
			GroupName: "Home",
			Timezone:  "Europe/Madrid",
			StartDate: time.Now().Format(config.DateLayout),
			Members:   []string{"Ana", "Bruno", "Carla", "Diego", "Elena"},
		}
		if err := config.Save(path, defaultCfg); err != nil {
			return nil, fmt.Errorf("creating default config: %w", err)
		}
		log.Printf("no config found at %s, created a default one", path)
		return defaultCfg, nil
	}
	return config.Load(path)
}

func printStartupInfo(port string) {
	log.Printf("MyTurn %s running on:", version)
	log.Printf("  http://localhost:%s", port)
	for _, ip := range lanIPs() {
		log.Printf("  http://%s:%s", ip, port)
	}
}

// lanIPs returns non-loopback IPv4 addresses of this machine, best-effort.
func lanIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil {
			continue
		}
		ips = append(ips, ip4.String())
	}
	return ips
}
