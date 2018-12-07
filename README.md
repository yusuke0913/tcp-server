# Setup
This project should be located $GOPATH/src/github.com/yusuke0913/tcp-server
```sh
$GOPATH/src/github.com/yusuke0913/tcp-server
```

```sh
make dep
```

I've not used any external libs.
localhost:7000 is used for this server.

# Directories

```
.
├── bin
│   ├── server
│   └── client
├── server
│   ├── main.go
│   └── main_test.go
├── client 
│   ├── main.go
│   └── main_test.go
├── api
│   ├── api.go
```

### bin directory
has a binary built.

### api directory
has message structs between server and client.

### server directory
has a server source code and testing code.

### client directory
has a client source code and testing code.

# Build

```sh
make server
make client
```

# Run

```sh
./bin/server
./bin/cleint
```

## Client Concurrency
The default number of clients is 10.
You can update the concurrency of clients by command line arguments.

```sh
./bin/client 30
```

# Testing

```sh
make server-test
make client-test
```

server testing doesn't work if a server is running.
client testing doesn't work if a server doesn't run.
