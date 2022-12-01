.PHONY: clean

clean:
	@go clean
	@rm ./target/levelman

build_linux:
	@go fmt
	@GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Commit=$(shell git rev-parse HEAD) -X main.Time=$(shell date --iso=seconds) -s -w" -o ./target/levelman_linux_amd64
	@upx --best --lzma ./target/levelman_linux_amd64

build_windows:
	@go fmt
	@GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Commit=$(shell git rev-parse HEAD) -X main.Time=$(shell date --iso=seconds) -s -w" -o ./target/levelman_win_amd64.exe
	@upx --best --lzma ./target/levelman_win_amd64.exe

build_darwin:
	@go fmt
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Commit=$(shell git rev-parse HEAD) -X main.Time=$(shell date --iso=seconds) -s -w" -o ./target/levelman_darwin_arm64
	@upx --best --lzma ./target/levelman_darwin_arm64

build_all: build_linux build_darwin build_windows

run: build_linux
	@./target/levelman