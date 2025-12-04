// Package config provides configuration management for the router worker.
//
// Configuration is loaded from environment variables and validated on startup.
// All configuration options have sensible defaults for development.
//
// Example usage:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(cfg)
package config
