package main

import (
	"context"
	"log"
	"net"
	"testing"
)

func TestRunServer(t *testing.T) {

	chErrorSv := make(chan error)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		err := runServer(ctx, network, address)
		if err != nil {
			chErrorSv <- err
			t.Fatalf("Faild	RunServer	%#v", err)
		}
		cancel()
	}()

	go func() {
		_, err := net.Dial(network, address)
		if err != nil {
			t.Fatalf("Faild	Connect	%#v", err)
		}
		log.Println("Connected to Server!")
		cancel()
	}()

	for {
		select {
		case err := <-chErrorSv:
			t.Fatalf("Faild	Connect	%#v", err)
			return
		case <-ctx.Done():
			log.Println("Done	TestRunServer")
			return
		}
	}
}
