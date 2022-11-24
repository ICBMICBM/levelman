.PHONY: clean

clean:
	@go clean
	@rm ./target/levelman

build_linux:
	@go fmt
	@GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Commit=$(shell git rev-parse HEAD) -X main.Time=$(shell date --iso=seconds) -s -w" -o ./target/levelman
	@upx --best --lzma ./target/levelman

build_windows:
	@go fmt
	@GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Commit=$(shell git rev-parse HEAD) -X main.Time=$(shell date --iso=seconds) -s -w" -o ./target/levelman.exe
	@upx --best --lzma ./target/levelman.exe

build_darwin:
	@go fmt
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Commit=$(shell git rev-parse HEAD) -X main.Time=$(shell date --iso=seconds) -s -w" -o ./target/levelman
	@upx --best --lzma ./target/levelman

run: build_linux
	@./target/levelman