# Changelog

All notable changes to the DA Node Router will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial router worker implementation
- Three routing modes: deterministic (CEL), LLM, and hybrid
- Redis Streams integration for work distribution
- Health check endpoint for Kubernetes
- Horizontal scaling support via consumer groups
- CEL expression evaluator for deterministic routing
- Handlebars template engine for LLM prompts
- Anthropic Claude integration for LLM routing
- Graceful shutdown handling
- Docker multi-platform support (amd64, arm64)
- GitHub Actions CI/CD pipelines
- Comprehensive documentation

### Configuration
- Environment-based configuration
- Support for multiple LLM providers (Anthropic primary)
- Configurable worker ID for multi-instance deployments
- Redis connection pooling

### Documentation
- Architecture and internals guide
- Routing strategies guide with examples
- API reference
- Configuration guide
- Deployment guide

## [0.1.0] - TBD

### Added
- MVP release
- Core routing functionality
- Basic monitoring and health checks

[Unreleased]: https://github.com/aescanero/dago-node-router/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/aescanero/dago-node-router/releases/tag/v0.1.0
