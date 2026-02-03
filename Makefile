BIN_DIR := bin
BINARY := klyr

.PHONY: build test lint fmt demo clean

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/klyr

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

demo:
	docker compose -f demo/compose.yaml up --build

clean:
	rm -rf ./bin ./logs ./state
