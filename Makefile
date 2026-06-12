.PHONY: build test run-seed run-agent clean docker

build:
	go build -o bin/seed ./cmd/seed
	go build -o bin/agent ./cmd/agent
	go build -o bin/founder ./cmd/founder
	go build -o bin/gateway ./cmd/gateway

test:
	go test ./... -v -count=1 -timeout 120s

run-seed:
	go run ./cmd/seed -port 4001 -data ./seed-data

run-agent:
	go run ./cmd/agent -port 4002 -data ./agent-data

clean:
	rm -rf bin/ seed-data agent-data founder-data

docker:
	docker build -t musketeers -f docker/Dockerfile .
