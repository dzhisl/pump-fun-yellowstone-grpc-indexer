# Pump.fun Indexer

A high-performance indexer for tracking Pump.fun token creation and swap transactions on Solana.

## Features

- Real-time monitoring of Pump.fun transactions (token creation and swaps)
- Efficient batch processing of swap transactions (50 per batch)
- PostgreSQL database storage with connection pooling
- Configurable logging levels
- GRPC custom [transaction converter](https://github.com/dzhisl/geyser-converter) integration
- Docker-compose setup for easy deployment

## Prerequisites

- Docker and Docker Compose
- Go 1.23 or higher (if running locally)
- PostgreSQL (included in Docker setup)
- GRPC endpoint for Solana transactions

## Getting Started

### Using Docker (Recommended)

1. Clone this repository
2. Change GRPC endpoint in `docker-compose.yml` :
3. Run the application:
   ```bash
   docker-compose up -d
   ```

### Running Locally

1. Clone this repository
2. Set up a PostgreSQL database
3. Add envs:
   ```
   DB_HOST=localhost
   DB_USER=admin
   DB_PASSWORD=admin
   DB_NAME=indexer
   DB_PORT=5432
   DB_SSLMODE=disable
   LOG_LEVEL=info
   GRPC_ENDPOINT=your_grpc_endpoint_url
   ```
4. Run:
   ```bash
   go run cmd/main.go
   ```

## Database Schema

![alt text](https://i.imgur.com/wsfmYVk.png)

## Architecture

The indexer follows this workflow:

1. Connects to a GRPC stream for Solana transactions
2. Filters transactions for the Pump.fun program (`6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P`)
3. Parses transaction instructions to identify:
   - Token creation events (14 accounts)
   - Swap events (12 accounts)
4. Processes data in batches for efficiency
5. Stores data in PostgreSQL with proper indexing

## Contributing

Contributions are welcome! Please open an issue or pull request with your suggestions or improvements.
