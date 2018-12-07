SERVER_OUT := "bin/server"
CLIENT_OUT := "bin/client"

PKG := "github.com/yusuke0913/tcp-server"
SERVER_PKG_BUILD := "${PKG}/server"
CLIENT_PKG_BUILD := "${PKG}/client"

CLIENT_TEST_DIR := "client"
SERVER_TEST_DIR := "server"

all: server client

dep: ## Get the dependencies
	@dep ensure

server: dep api ## Build the binary file for server
	@go build -i -v -o $(SERVER_OUT) $(SERVER_PKG_BUILD)

client: dep api ## Build the binary file for client
	@go build -i -v -o $(CLIENT_OUT) $(CLIENT_PKG_BUILD)

clean: ## Remove previous builds
	@rm $(SERVER_OUT) $(CLIENT_OUT)

client-test: ## Remove previous builds
	# @go test $(CLIENT_PKG_BUILD)/... -v
	@cd ${CLIENT_TEST_DIR}; pwd; go test -v

server-test: ## Remove previous builds
	@cd ${SERVER_TEST_DIR}; pwd; go test -v

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

