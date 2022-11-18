.PHONY: clean

clean:
	@go clean
	@rm ./target/levelman

build:
	@go fmt
	@go build -ldflags="-X main.Commit=$(shell git rev-parse HEAD) -X main.Time=$(shell date --iso=seconds)" -o ./target/levelman

run: build
	@./target/levelman