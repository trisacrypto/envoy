package main

// Ensures comments for Swagger are proper
//go:generate go run github.com/swaggo/swag/cmd/swag@latest fmt

// Ensures that the Swagger files exist
//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -g ./cmd/envoy/main.go --parseInternal --parseDependency
