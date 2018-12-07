# TCP Server Client samples
Implement a tcp server and client. Messages structs is exchanged by gob library.
Client has three kind of messages to relay text messages between other clients.

# Setup
This project should be located $GOPATH/src/github.com/yusuke0913/tcp-server
```sh
$GOPATH/src/github.com/yusuke0913/tcp-server
```

```sh
make dep
```

# Server Listen Port
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
It has built binaries

### api directory
It has message structs and interfaces between server and client.

### server directory
It has the server source code and testing code.

### client directory
It has the client source code and testing code.

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
The default number of clients is 3.
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
