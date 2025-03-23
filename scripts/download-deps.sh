#!/bin/bash

# Script to download all required dependencies

echo "Downloading dependencies..."

# Core dependencies
go get github.com/joho/godotenv
go get github.com/google/uuid
go get github.com/go-redis/redis/v8
go get github.com/confluentinc/confluent-kafka-go/v2/kafka
go get github.com/btcsuite/btcd/btcec/v2
go get github.com/btcsuite/btcd/btcec/v2/ecdsa
go get github.com/btcsuite/btcutil/base58

# API dependencies
go get github.com/go-chi/chi/v5
go get github.com/go-chi/chi/v5/middleware
go get github.com/go-chi/cors
go get github.com/go-chi/httprate
go get github.com/go-chi/jwtauth/v5

echo "Dependencies downloaded successfully!"
