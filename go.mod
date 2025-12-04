module github.com/aescanero/dago-node-router

go 1.25.5

require (
	github.com/aescanero/dago-libs v0.2.0

	// LLM client
	github.com/anthropics/anthropic-sdk-go v1.17.0

	// Template engine (LLM prompts)
	github.com/aymerick/raymond v2.0.2+incompatible

	// Configuration
	github.com/caarlos0/env/v10 v10.0.0

	// CEL evaluator (deterministic routing)
	github.com/google/cel-go v0.18.2

	// Redis Streams for events
	github.com/redis/go-redis/v9 v9.3.0

	// Logging
	go.uber.org/zap v1.26.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect

	// UUID
	github.com/google/uuid v1.5.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240722135656-d784300faade // indirect
)
