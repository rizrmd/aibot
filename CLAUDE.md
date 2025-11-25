# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a sophisticated Go-based cryptocurrency trading bot that combines grid trading with breakout detection. The bot features advanced risk management, false breakout protection, and comprehensive streaming capabilities with replay functionality.

**Architecture**: Clean Architecture with hexagonal design, strong separation of concerns, and interface-driven development.

## Common Development Commands

### Essential Workflow
```bash
# Set up development environment (first time only)
make dev

# Create default config file
make config

# Build and run normally
make run

# Build and run with debug mode
make run-debug

# Run with hot reload during development
make watch
```

### Testing and Quality Assurance
```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run race condition tests
make test-race

# Full quality check (format, vet, lint, test)
make check

# Run security checks
make security

# Check for vulnerabilities
make vuln-check
```

### Build and Deployment
```bash
# Build for current platform
make build

# Build for all platforms (release)
make build-release

# Install globally
make install

# Build Docker image
make docker-build
```

### Historical Replay
```bash
# Run with historical data replay (via streaming config)
make run

# Configure replay mode in config.json:
# {
#   "stream": {
#     "provider_type": "replay",
#     "data_source": {
#       "type": "file",
#       "directory": "./historical_data",
#       "start_time": "2024-01-01T00:00:00Z",
#       "end_time": "2024-12-31T23:59:59Z"
#     }
#   }
# }
```

## High-Level Architecture

### Core Components

**Bot Orchestrator** (`internal/bot/orchestrator.go`): Central coordinator that manages trading modes (Grid, Breakout, Recovery, Stability, Idle) and handles intelligent strategy transitions based on market conditions.

**Strategy Engine** (`internal/strategy/`): Modular trading strategies with clear separation:
- Grid Trading: Fee-aware grid spacing with dynamic bounds
- Breakout Detection: Multi-factor confirmation (volume, momentum, RSI, ATR)
- False Breakout Protection: Pattern recognition and recovery mechanisms
- Price Stability Detection: Multi-timeframe analysis

**Configuration System** (`internal/config/config.go`): JSON-based configuration with comprehensive validation and environment variable overrides.

**Streaming Interface** (`pkg/stream/interface.go`): Provider pattern for data streaming with replay provider for testing and ready for live exchange integration.

**Trading Execution** (`pkg/trading/interface.go`): Executor pattern for abstract trading execution across different platforms with realistic order execution.

### Key Design Patterns

1. **Provider Pattern**: Abstract data streaming and trading execution
2. **Strategy Pattern**: Pluggable trading strategies
3. **Observer Pattern**: Event-driven architecture for market data
4. **Factory Pattern**: Component creation and dependency injection

### Domain Types (`internal/types/`)

Well-defined domain models including:
- Grid management (levels, bounds, state)
- Order lifecycle management
- Position tracking and PnL calculation
- Market data structures (OHLCV, ticker)

### Entry Point

`cmd/main.go`: Application initialization with proper signal handling, context management for graceful shutdown, and component orchestration.

## Configuration

The bot uses `config.json` with these key sections:
- **trading**: Initial balance, leverage, fees, symbols
- **strategy**: Grid levels, breakout confirmation, stability thresholds
- **risk**: Portfolio risk limits (5% max), position risk (2% max), drawdown limits
- **stream**: Data provider configuration (live by default)

Run `make config` to create a default configuration file.

## Development Guidelines

### Code Organization
- Follow Go package naming conventions
- Keep interfaces small and focused
- Use dependency injection for testability
- Maintain clear separation between domain and infrastructure code

### Testing Strategy
- Unit tests for all core components
- Integration tests for cross-component functionality
- Race condition testing with `-race` flag
- Benchmark tests for performance-critical paths

### Error Handling
- Use explicit error returns throughout
- Implement graceful degradation for component failures
- Include retry logic with circuit breakers
- Comprehensive logging with structured format

### Performance Considerations
- Concurrent processing with goroutines
- Efficient data structures and buffering
- Sub-second decision making capabilities
- Configurable resource limits

## Important Notes

- **Live Mode**: Default configuration runs in live mode (with testnet support) for real trading
- **Risk Management**: Built-in position sizing (2% max per position, 5% portfolio)
- **False Breakout Protection**: Advanced pattern recognition prevents false signals
- **Multi-Timeframe Analysis**: Uses 1s, 3s, and 15s candle aggregation from 300ms data
- **Fee Awareness**: Grid spacing accounts for maker/taker fees (0.02%/0.06%)

## Current State

The codebase is in production-ready state with comprehensive documentation, complete feature set, and professional software engineering practices. Recent commit shows initial implementation with full functionality.