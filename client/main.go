package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/yusuke0913/tcp-server/api"
)

const (
	network              = "tcp"
	address              = "localhost:7000"
	keepAliveSeconds     = 10
	deadLineTimeDuration = time.Second * 10
)

var (
	clientNum = 3
	handleMap = map[uint8]handler{
		api.ResponseTypeIdentify:  handleIdentify,
		api.ResponseTypeList:      handleList,
		api.ResponseTypeRelay:     handleRelay,
		api.ResponseTypeRelayDone: handleRelayDone,
	}
)

func main() {

	log.Printf("args	len:%d	args:%v", len(os.Args), os.Args)
	if len(os.Args) >= 2 {
		concurrency, err := strconv.Atoi(os.Args[1])
		if err != nil {
			clientNum = concurrency
		}
		clientNum = concurrency
		log.Printf("clientNum:%d	typeof:%v	err:%v", clientNum, reflect.TypeOf(concurrency), err)
	}

	api.Register()

	ctx := context.Background()
	// ctx, cancel := context.WithDeadline(ctx, time.Now().Add(deadLineTimeDuration))
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	for i := 0; i < clientNum; i++ {
		wg.Add(1)
		go func(cnum int) {
			client, err := NewClient(ctx, network, address)
			if err != nil {
				log.Printf("FAILD_NEW_CLIENT	i:%d", cnum)
				return
			}
			log.Printf("NEW_CLIENT	i:%d	userID:%d", cnum, client.userID)

			defer func() {
				wg.Done()
				log.Printf("EXIT_CLIENT	i:%d	userID:%d", cnum, client.userID)
			}()

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
			}

		}(i)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		<-signals
		cancel()
	}()

	wg.Wait()
	log.Printf("Done")
}
