# Historical Data Directory

This directory contains historical OHLCV data files for backtesting.

## Required CSV Format

CSV files must have the following columns in order:
```
timestamp,open,high,low,close,volume
```

## Supported Timestamp Formats

- `YYYY-MM-DD HH:MM:SS` (e.g., 2024-01-01 00:00:00)
- `YYYY-MM-DDTHH:MM:SSZ` (e.g., 2024-01-01T00:00:00Z)
- `YYYY-MM-DDTHH:MM:SS.000Z` (e.g., 2024-01-01T00:00:00.000Z)
- `YYYY/MM/DD HH:MM:SS` (e.g., 2024/01/01 00:00:00)

## File Naming

Files can be named in any of these formats:
- `BTCUSDT.csv`
- `btcusdt.csv`
- `BTCUSDT_1m.csv`

## Example Data

```csv
timestamp,open,high,low,close,volume
2024-01-01 00:00:00,45000.00,45200.00,44800.00,45100.00,1500.25
2024-01-01 00:01:00,45100.00,45300.00,44900.00,45200.00,1200.50
2024-01-01 00:02:00,45200.00,45400.00,45000.00,45300.00,1800.75
```

## Data Sources

You can obtain historical data from:
- Binance API (https://binance-docs.github.io/apidocs/spot/en/#kline-candlestick-data)
- CoinGecko API
- CCXT library
- CryptoCompare API

## Data Validation

The system will validate:
- Required columns exist
- OHLC relationships (high >= open/close/low, low <= open/close/high)
- Numeric values can be parsed
- Timestamps are in valid format

Place your historical CSV files in this directory before running backtests.