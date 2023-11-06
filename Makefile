.PHONY: all golang-build yarn-install golang-test yarn-test test clean

all: golang-install golang-build yarn-install

golang-install:
	@echo "Ensuring Golang dependencies..."
	go mod download

golang-build: golang-install
	@echo "Building Golang application..."
	go build

yarn-install:
	@echo "Installing yarn dependencies..."
	yarn

golang-test: golang-install
	@echo "Running Golang tests..."
	go test ./...

yarn-test:
	@echo "Running yarn tests..."
	npm run test

test: golang-test yarn-test

clean:
	@echo "Cleaning up..."
	go clean
	rm -rf node_modules && rm -f yarn.lock
