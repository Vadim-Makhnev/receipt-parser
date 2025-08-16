build:
	@go build -o ./bin/main.exe ./cmd/main.go

run: build
	@./bin/main.exe