Signals
=======

Overview
--------

The Signals package provides an algorithmic trading system for optimizing and
executing trade signals based on historical market data. It supports genetic
algorithm-based optimization and deep backtesting to refine trading strategies.

Supported Platforms & Executables
---------------------------------

Signals is available for multiple operating systems with the following executables:

Windows: signals-windows-amd64.exe
macOS: signals-darwin-arm64
Linux:
  signals-linux-arm64
  signals-linux-amd64

Installation
------------

Download the appropriate executable for your operating system.

Place the executable in a directory of your choice.

Ensure you have configured the .env.local file with the necessary parameters (see below).

Configuration
-------------

The Signals package requires environment variables to be set in a file named
.env.local. Below is a sample configuration:

# OKX API Configuration
OKX_BASE_URL=https://www.okx.com
OKX_API_KEY=
OKX_API_SECRET=
OKX_API_PASSPHRASE=

# Trading Strategy Configuration
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

# RSI Trading Parameters
SIGNALS_RSI_UPPER_BOUND=60
SIGNALS_RSI_LOWER_BOUND=40

# Optimizer Configuration
SIGNALS_OPTIMIZER_POPULATION_SIZE=75
SIGNALS_OPTIMIZER_GENERATIONS=50
SIGNALS_OPTIMIZER_RETAIN_RATE=0.45
SIGNALS_OPTIMIZER_MUTATION_RATE=0.25
SIGNALS_OPTIMIZER_ELITE_COUNT=5

Commands
--------

Run the Signals package using the following commands:

1. Optimize Trading Strategy

Runs the genetic algorithm to evolve the best trading strategy based on historical data.

./signals optimize

2. Train & Backtest a Model

Trains a basic model using the configured parameters, performs deep backtesting, and
prints out model statistics.

./signals train

Notes
-----

Ensure that the .env.local file is properly configured before running any commands.

API keys for OKX are required for live trading; ensure these are securely stored.

The optimizer runs a genetic algorithm for natural selection of trading strategies.

The train command performs a detailed evaluation of the trading model, allowing for
fine-tuning before deployment.

Support
-------

For issues, bug reports, or feature requests, please contact the project maintainers
or submit an issue in the project repository.

