# Signals Package

## Overview

The **Signals** package is a trading bot that optimizes and executes strategies based on market signals. It uses a natural selection algorithm to evolve trading models and supports multiple platforms.

## Installation

### Executable Files
Download the appropriate executable for your operating system:

- **Windows**: `signals-windows-amd64.exe`
- **macOS**: `signals-darwin-arm64`
- **Linux**:
  - ARM64: `signals-linux-arm64`
  - AMD64: `signals-linux-amd64`

## Configuration

Before running the bot, configure the `.env.local` file with your preferred settings.

### Example `.env.local` Configuration:

```ini
# OKX API Configuration
OKX_BASE_URL=https://www.okx.com
OKX_API_KEY=
OKX_API_SECRET=
OKX_API_PASSPHRASE=

# General Trading Configuration
SIGNALS_GENERATIONS=3
SIGNALS_GENERATIONS_DURATION=3600
SIGNALS_INSTRUMENT=DOGE-USDT-SWAP
SIGNALS_CANDLES=5
SIGNALS_TAKE_PROFIT=0.4
SIGNALS_STOP_LOSS=0.1
SIGNALS_LEVERAGE=50
SIGNALS_TRADE_MULTIPLIER=1
SIGNALS_COMMISSION=0.001
SIGNALS_COOLDOWN=300

# Trading Strategy Parameters
SIGNALS_RSI_UPPER_BOUND=60
SIGNALS_RSI_LOWER_BOUND=40

# Optimizer Configuration
SIGNALS_OPTIMIZER_POPULATION_SIZE=75
SIGNALS_OPTIMIZER_GENERATIONS=50
SIGNALS_OPTIMIZER_RETAIN_RATE=0.45
SIGNALS_OPTIMIZER_MUTATION_RATE=0.25
SIGNALS_OPTIMIZER_ELITE_COUNT=5
```

### Binance Candlestick Data

By default Signals will use OKX candlestick data. To switch to Binance
candlestick data use the following in your env file:

```ini
SIGNALS_NETWORK=binance
SIGNALS_INSTRUMENT=DOGEUSDT
```

## Usage

### Running the Optimizer

To run the natural selection algorithm for optimizing trading models:

```sh
./signals optimize
```

The optimizer will output a CSV file containing useful stats from each generation, including
the best strategy (by fitness score) and its parameters from that generation. Each generation
outputs to the console useful stats, including a copy and pastable set of model parameters 
for the best strategy in that generation.

### Training a Basic Model

To train a model using the configured parameters and perform deep backtesting:

```sh
./signals train
```

This will output model statistics, including fitness scores and backtest results.

## License

This package is provided as-is with no warranty express or implied whatsoever. Ensure you configure API keys securely and trade responsibly.