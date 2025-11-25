# AI Trading Bot

A sophisticated cryptocurrency trading bot that combines grid trading with breakout detection, featuring advanced risk management, false breakout protection, and comprehensive streaming capabilities with historical replay.

## 🚀 Features

### Core Trading Strategies
- **Grid Trading**: Automated grid trading with fee-aware spacing optimization
- **Breakout Detection**: Multi-factor breakout confirmation with technical indicators
- **False Breakout Protection**: Advanced pattern recognition to avoid false signals
- **Position Management**: Deterministic position sizing and risk-based stop losses

### Advanced Analytics
- **Technical Analysis**: RSI, MACD, ATR, Bollinger Bands integration
- **Multi-Timeframe Analysis**: 1s, 3s, and 15s candle aggregation from 300ms data
- **Price Stability Detection**: Algorithmic detection of market stability conditions
- **Risk Management**: Portfolio-wide risk assessment with VaR and stress testing

### Performance & Risk
- **Fee Optimization**: Grid spacing calculations account for maker/taker fees (0.02%/0.06%)
- **Position Sizing**: Risk-based position sizing with confidence adjustments
- **Drawdown Protection**: Automatic position reduction on excessive drawdowns
- **Correlation Analysis**: Multi-asset correlation risk management

### Infrastructure
- **Real-time Data**: WebSocket streaming with 300ms update intervals
- **Replay Environment**: Comprehensive historical data replay with realistic trading simulation
- **Configuration Management**: JSON-based configuration with validation
- **Structured Logging**: Comprehensive logging with performance metrics

## 📋 Requirements

- Go 1.19 or higher
- 700+ cryptocurrency trading pairs (WebSocket data)
- Historical OHLCV data (300ms intervals)

## 🛠️ Installation

### Quick Start
```bash
# Clone the repository
git clone https://github.com/yourusername/ai-trading-bot.git
cd ai-trading-bot

# Install dependencies
make deps

# Create default configuration
make config

# Run the bot
make run
```

### Build from Source
```bash
# Build for your platform
make build

# Build for multiple platforms
make build-release

# Install globally
make install
```

### Development Setup
```bash
# Set up development environment
make dev

# Install development tools
make tools

# Run with hot reload
make watch
```

## ⚙️ Configuration

The bot uses a JSON configuration file (`config.json`) that is automatically created with default values on first run.

### Key Configuration Sections

#### Trading Settings
```json
{
  "trading": {
    "initial_balance": 10000.0,
    "default_leverage": 5.0,
    "max_leverage": 10.0,
    "maker_fee": 0.0002,
    "taker_fee": 0.0006,
    "supported_symbols": ["BTCUSDT", "ETHUSDT"],
    "default_symbol": "BTCUSDT"
  }
}
```

#### Strategy Parameters
```json
{
  "strategy": {
    "grid": {
      "min_grid_levels": 10,
      "max_grid_levels": 30,
      "min_profit_per_level": 0.0015
    },
    "breakout": {
      "confirmation_candles": 3,
      "min_breakout_strength": 0.003
    },
    "stability": {
      "analysis_window": 10,
      "volatility_threshold": 0.005
    }
  }
}
```

#### Risk Management
```json
{
  "risk": {
    "max_portfolio_risk": 0.05,
    "max_position_risk": 0.02,
    "max_drawdown": 0.10,
    "concentration_limit": 0.3
  }
}
```

## 🎯 Trading Strategy

### Grid Trading Mode
- **Setup**: Historical analysis determines optimal grid parameters
- **Spacing**: Fee-aware calculation ensures profitability (minimum 0.15% per level)
- **Levels**: 10-30 grid levels based on market volatility
- **Bounds**: Dynamic grid bounds with ATR-based expansion

### Breakout Detection
- **Confirmation**: 3-candle confirmation period (900ms at 300ms intervals)
- **Multi-Factor**: Volume, momentum, RSI, and ATR confirmation
- **Confidence Scoring**: Weighted confidence calculation for signal reliability

### False Breakout Protection
- **Pattern Recognition**: Quick reversals, volume drops, momentum shifts
- **Recovery Actions**: Deterministic actions for different false breakout types
- **Risk Mitigation**: Automatic position reduction on high false breakout probability

### Stability Detection
- **Multi-Timeframe**: Primary (3s) and secondary (15s) analysis
- **Volatility Analysis**: Adaptive volatility thresholds
- **Range Contraction**: Detects price consolidation patterns

## 📊 Historical Replay

### Running Historical Replay
```bash
# Run with historical data replay
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

### Replay Features
- **Historical Data**: CSV format OHLCV data support
- **Realistic Trading**: Commission, slippage, and latency modeling
- **Performance Metrics**: Real-time P&L tracking and risk monitoring
- **Trade Export**: Detailed trade logs in CSV format
- **Configurable Speed**: Accelerated or real-time replay
- **Multi-timeframe**: Automatic candle aggregation

### Data Format
CSV files should have the following format:
```csv
timestamp,open,high,low,close,volume
2023-01-01 00:00:00,45000.0,45200.0,44800.0,45100.0,1234.5
```

## 🔧 Usage

### Command Line Options
```bash
# Run with default config
./trading-bot

# Use custom configuration
./trading-bot -config /path/to/config.json

# Enable debug mode
./trading-bot -debug

# Show version
./trading-bot -version

# Show help
./trading-bot -help
```

### Environment Variables
```bash
# Configuration file path
export TRADING_BOT_CONFIG_PATH="/path/to/config.json"

# Debug mode
export TRADING_BOT_DEBUG="true"

# Log level override
export TRADING_BOT_LOG_LEVEL="debug"

# Environment setting
export TRADING_BOT_ENVIRONMENT="production"
```

### Control Commands
```bash
# Send control commands (if supported by your implementation)
echo '{"type": "pause"}' | nc localhost 8080
echo '{"type": "resume"}' | nc localhost 8080
echo '{"type": "switch_mode", "payload": "grid"}' | nc localhost 8080
```

## 📈 Monitoring & Logging

### Structured Logging
The bot provides comprehensive logging with the following components:
- **Trading**: Trade executions, position updates
- **Strategy**: Signal generation, mode transitions
- **Risk**: Risk alerts, limit breaches
- **Performance**: PnL tracking, win rates
- **System**: Component health, errors

### Performance Metrics
```json
{
  "total_trades": 150,
  "win_rate": 0.68,
  "total_pnl": 1250.50,
  "sharpe_ratio": 1.85,
  "max_drawdown": 0.08,
  "current_mode": "grid",
  "active_positions": 3
}
```

### Log Formats
- **JSON**: Structured logs for machine processing
- **Text**: Human-readable logs for debugging
- **File Rotation**: Automatic log rotation and compression

## 🧪 Testing

### Running Tests
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run race condition tests
make test-race

# Run benchmarks
make benchmark
```

### Code Quality
```bash
# Format code
make fmt

# Run linter
make lint

# Run security checks
make security

# Run vulnerability checks
make vuln-check

# Full quality check
make check
```

## 🔒 Security

### Risk Management
- **Position Sizing**: Maximum 2% risk per position
- **Portfolio Risk**: Maximum 5% total portfolio risk
- **Leverage Limits**: Configurable maximum leverage
- **Margin Protection**: Automatic position reduction on margin calls

### Security Features
- **Input Validation**: Comprehensive configuration validation
- **Error Handling**: Graceful error handling with recovery
- **Audit Logging**: Complete trade and configuration audit trail
- **Access Control**: Secure API endpoints (if applicable)

## 📚 API Documentation

### Core Components

#### Streaming Interface
```go
type Provider interface {
    Start(ctx context.Context, symbols []string) error
    Stop() error
    GetTickerChannel() <-chan *types.Ticker
    GetOHLCVChannel() <-chan *types.OHLCV
}
```

#### Trading Interface
```go
type Executor interface {
    Connect(ctx context.Context) error
    OpenLong(symbol string, quantity, price float64) (*types.OrderResult, error)
    OpenShort(symbol string, quantity, price float64) (*types.OrderResult, error)
    ClosePosition(symbol string, quantity float64) (*types.OrderResult, error)
}
```

#### Strategy Components
- **Grid Setup**: `strategy.GridSetup`
- **Breakout Detection**: `strategy.BreakoutDetector`
- **False Breakout Detection**: `strategy.FalseBreakoutDetector`
- **Price Stability**: `strategy.PriceStabilityDetector`
- **Risk Management**: `strategy.RiskManager`

## 🤝 Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and quality checks
5. Submit a pull request

### Code Standards
- Follow Go formatting conventions (`gofmt`)
- Add comprehensive tests
- Update documentation
- Ensure backward compatibility

### Submitting Issues
Please include:
- Go version
- Operating system
- Configuration (sanitized)
- Error messages
- Steps to reproduce

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Indicator Go Library](https://github.com/cinar/indicator) - Technical analysis indicators
- [Logrus](https://github.com/sirupsen/logrus) - Structured logging
- [Lumberjack](https://github.com/natefinch/lumberjack) - Log rotation

## 📞 Support

- **Documentation**: Check this README and inline code documentation
- **Issues**: [GitHub Issues](https://github.com/yourusername/ai-trading-bot/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/ai-trading-bot/discussions)

## 🗺️ Roadmap

### Upcoming Features
- [ ] Live exchange integrations (Binance, Bybit, etc.)
- [ ] Web dashboard for monitoring
- [ ] Mobile app for alerts
- [ ] Advanced optimization algorithms
- [ ] Machine learning integration
- [ ] Social trading features

### Version History
- **v1.0.0**: Initial release with core grid and breakout trading
- **v1.1.0**: Enhanced streaming and historical replay capabilities
- **v1.2.0**: Multi-exchange support (planned)

---

**⚠️ Disclaimer**: This software is for educational and research purposes. Trading cryptocurrencies involves substantial risk of loss. Use at your own risk and never invest more than you can afford to lose.