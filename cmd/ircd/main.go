package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/supamanluva/ircd/internal/logger"
	"github.com/supamanluva/ircd/internal/server"
)

var (
	configPath = flag.String("config", "config/config.yaml", "Path to configuration file")
	version    = "0.1.0"
)

func main() {
	flag.Parse()

	// Initialize logger
	log := logger.New()
	log.Info("Starting IRC Server", "version", version)

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create server instance
	srv, err := server.New(cfg, log)
	if err != nil {
		log.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Info("Received shutdown signal, shutting down gracefully...")
		cancel()
		srv.Shutdown()
	case err := <-errChan:
		log.Error("Server error", "error", err)
		cancel()
		srv.Shutdown()
		os.Exit(1)
	}

	log.Info("Server stopped")
}

func loadConfig(path string) (*server.Config, error) {
	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		// Return defaults if config file doesn't exist
		return &server.Config{
			ServerName:   "IRCServer",
			Host:         "0.0.0.0",
			Port:         6667,
			MaxClients:   1000,
			TLSEnabled:   false,
			TLSPort:      6697,
			TLSCertFile:  "certs/server.crt",
			TLSKeyFile:   "certs/server.key",
			PingInterval: 60,
			Timeout:      300,
		}, nil
	}

	// Parse YAML
	var configData struct {
		Server struct {
			Name         string `yaml:"name"`
			Host         string `yaml:"host"`
			Port         int    `yaml:"port"`
			MaxClients   int    `yaml:"max_clients"`
			Timeout      int    `yaml:"timeout_seconds"`
			PingInterval int    `yaml:"ping_interval_seconds"`
			TLS          struct {
				Enabled  bool   `yaml:"enabled"`
				Port     int    `yaml:"port"`
				CertFile string `yaml:"cert_file"`
				KeyFile  string `yaml:"key_file"`
			} `yaml:"tls"`
		} `yaml:"server"`
		WebSocket struct {
			Enabled        bool     `yaml:"enabled"`
			Host           string   `yaml:"host"`
			Port           int      `yaml:"port"`
			AllowedOrigins []string `yaml:"allowed_origins"`
			TLS            struct {
				Enabled  bool   `yaml:"enabled"`
				CertFile string `yaml:"cert_file"`
				KeyFile  string `yaml:"key_file"`
			} `yaml:"tls"`
		} `yaml:"websocket"`
		Linking struct {
			Enabled     bool   `yaml:"enabled"`
			Host        string `yaml:"host"`
			Port        int    `yaml:"port"`
			ServerID    string `yaml:"server_id"`
			Description string `yaml:"description"`
			Password    string `yaml:"password"`
			Links       []struct {
				Name        string `yaml:"name"`
				SID         string `yaml:"sid"`
				Host        string `yaml:"host"`
				Port        int    `yaml:"port"`
				Password    string `yaml:"password"`
				AutoConnect bool   `yaml:"auto_connect"`
				IsHub       bool   `yaml:"is_hub"`
			} `yaml:"links"`
		} `yaml:"linking"`
		Operators []struct {
			Name     string `yaml:"name"`
			Password string `yaml:"password"`
		} `yaml:"operators"`
	}

	if err := yaml.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Build operators list
	operators := make([]server.Operator, len(configData.Operators))
	for i, op := range configData.Operators {
		operators[i] = server.Operator{
			Name:     op.Name,
			Password: op.Password,
		}
	}

	// Build links list
	links := make([]server.LinkConfig, len(configData.Linking.Links))
	for i, link := range configData.Linking.Links {
		links[i] = server.LinkConfig{
			Name:        link.Name,
			SID:         link.SID,
			Host:        link.Host,
			Port:        link.Port,
			Password:    link.Password,
			AutoConnect: link.AutoConnect,
			IsHub:       link.IsHub,
		}
	}

	// Build config
	config := &server.Config{
		ServerName:       configData.Server.Name,
		Host:             configData.Server.Host,
		Port:             configData.Server.Port,
		MaxClients:       configData.Server.MaxClients,
		TLSEnabled:       configData.Server.TLS.Enabled,
		TLSPort:          configData.Server.TLS.Port,
		TLSCertFile:      configData.Server.TLS.CertFile,
		TLSKeyFile:       configData.Server.TLS.KeyFile,
		PingInterval:     time.Duration(configData.Server.PingInterval) * time.Second,
		Timeout:          time.Duration(configData.Server.Timeout) * time.Second,
		Operators:        operators,
		WebSocketEnabled: configData.WebSocket.Enabled,
		WebSocketHost:    configData.WebSocket.Host,
		WebSocketPort:    configData.WebSocket.Port,
		WebSocketOrigins: configData.WebSocket.AllowedOrigins,
		WebSocketTLS:     configData.WebSocket.TLS.Enabled,
		WebSocketCert:    configData.WebSocket.TLS.CertFile,
		WebSocketKey:     configData.WebSocket.TLS.KeyFile,
		LinkingEnabled:   configData.Linking.Enabled,
		LinkingHost:      configData.Linking.Host,
		LinkingPort:      configData.Linking.Port,
		ServerID:         configData.Linking.ServerID,
		ServerDesc:       configData.Linking.Description,
		LinkPassword:     configData.Linking.Password,
		Links:            links,
	}

	// Set defaults for missing values
	if config.ServerName == "" {
		config.ServerName = "IRCServer"
	}
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}
	if config.Port == 0 {
		config.Port = 6667
	}
	if config.MaxClients == 0 {
		config.MaxClients = 1000
	}
	if config.TLSPort == 0 {
		config.TLSPort = 6697
	}
	if config.PingInterval == 0 {
		config.PingInterval = 60 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 300 * time.Second
	}
	if config.WebSocketPort == 0 {
		config.WebSocketPort = 8080
	}
	if config.WebSocketHost == "" {
		config.WebSocketHost = "0.0.0.0"
	}
	if len(config.WebSocketOrigins) == 0 {
		config.WebSocketOrigins = []string{"*"}
	}
	if config.LinkingPort == 0 {
		config.LinkingPort = 7777
	}
	if config.LinkingHost == "" {
		config.LinkingHost = "0.0.0.0"
	}

	return config, nil
}
