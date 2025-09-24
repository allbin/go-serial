# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go package for serial communication (`github.com/mdjarv/serial`) using Go 1.25.1. The project emphasizes clean, DRY code following Go best practices and idiomatic patterns.

## Development Commands

### Code Quality
- `go fmt ./...` - Format all Go files (required before commits)
- `go vet ./...` - Run static analysis
- `go build` - Build the package
- `go test ./...` - Run all tests
- `go test -v ./...` - Run tests with verbose output
- `go test -race ./...` - Run tests with race detection

### Module Management
- `go mod tidy` - Clean up module dependencies
- `go get <package>` - Add new dependencies
- `go mod download` - Download dependencies

## Architecture Principles

- **Clean Code**: Follow DRY principles, avoid code duplication
- **Idiomatic Go**: Use Go conventions and patterns
- **Best Practices**: Proper error handling, clear interfaces, good separation of concerns
- **Serial Communication Focus**: Design APIs around common serial port operations (open, read, write, configure)

## Code Standards

- All code must be formatted with `go fmt`
- Use meaningful variable and function names
- Implement proper error handling with descriptive messages
- Write tests for all public APIs
- Follow Go naming conventions (exported vs unexported)