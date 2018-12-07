package main

import (
	"bufio"
	"context"
	"encoding/gob"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yusuke0913/tcp-server/api"
)

const (
	network          = "tcp"
	address          = "localhost:7000"
	keepAliveSeconds = 10
	relayThreadNum   = 3
)

func main() {
	api.Register()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	for i := 0; i < relayThreadNum; i++ {
		wg.Add(1)
		go func(relayThreadNum int) {
			defer wg.Done()
			relay(ctx, relayThreadNum)
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		runServer(ctx, network, address)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		<-signals
		cancel()
	}()

	wg.Wait()
	log.Printf("DoneEverything")
}

func relay(ctx context.Context, threadNumber int) {

	for {
		select {
		case <-ctx.Done():
			// log.Printf("DONE_RELAY_GOROUTINE	threadNumber:%d", threadNumber)
			return
		case req := <-relayMessages:
			// log.Printf("START_RELAY	threadNumber:%d	%#v", threadNumber, req)
			fromUserID := req.UserID
			toUserIDs := req.ToUserIDs

			if len(toUserIDs) > api.MaxRelayMessageLength {
				log.Printf("RELAY	ERR	TOO_MANY_TO_USER_IDS	%#v", len(toUserIDs))
				break
			}

			res := api.RelayResponse{UserID: fromUserID, Message: req.Message}

			var count uint8
			for _, toUserID := range toUserIDs {

				if fromUserID == toUserID {
					continue
				}

				v, ok := sd.clientMap.Load(toUserID)
				if !ok {
					// log.Printf("RELAY_FAILD_TARGET_NOT_FOUND	threadNumber:%d	toUserID:%d", threadNumber, toUserID)
					continue
				}

				targetClient := v.(*client)
				err := targetClient.send(res)
				if err != nil {
					// log.Printf("RELAY_FAILD_SEND_TO_TARGET	threadNumber:%d	toUserID:%d	error:%#v", threadNumber, toUserID, err)
					sd.clientMap.Delete(toUserID)
					continue
				}
				count++
				// log.Printf("RELAY	toUserID:%d	res:%#v", toUserID, relayRes)
			}

			v, ok := sd.clientMap.Load(fromUserID)
			if !ok {
				// log.Printf("RELAY	NOT_FOUND	toUserID:%d", fromUserID)
				continue
			}
			doneRes := api.RelayDoneResponse{Count: count}
			fromClient := v.(*client)
			err := fromClient.send(doneRes)
			if err != nil {
				// log.Printf("RELAY_FAILED_DoneResponse	threadNumber:%d	fromUserID:%d	error:%#v", threadNumber, fromUserID, err)
				continue
			}
			// log.Printf("SEND_RESPONSE_RELAY_DONE	fromUserID:%d	res:%#v", fromUserID, res)
			// log.Printf("DONE_RELAY	threadNumber:%d	fromUserID:%d	relayCount:%d", threadNumber, fromUserID, count)
		}
	}
}

type sharedData struct {
	currentUserID uint64
	clientMap     sync.Map
}

var (
	sd        *sharedData
	handleMap = map[uint8]handler{
		api.RequestTypeIdentify: handleIdentify,
		api.RequestTypeList:     handleList,
		api.RequestTypeRelay:    handleRelay,
	}
	relayMessages = make(chan *api.RelayRequest)
)

type client struct {
	conn net.Conn

	mutex  sync.RWMutex
	reader *bufio.Reader

	wm     sync.Mutex
	writer *bufio.Writer

	userID uint64
}

func (c *client) send(res api.Responser) error {
	// log.Printf("SendLock	userID:%d	%#v", c.userID, *res)
	c.wm.Lock()
	defer c.wm.Unlock()

	var (
		n   int
		err error
	)

	bytes, err := api.Encode(res)
	if err != nil {
		return err
	}

	_, err = c.writer.Write(bytes)
	if err != nil {
		log.Printf("err	userID:%d	%d	%v", c.userID, n, err)
		return err
	}

	err = c.writer.Flush()
	if err != nil {
		return err
	}

	// log.Printf("Send	userID:%d	bytes:%d	res:%#v", c.userID, n, *res)
	return nil
}

func (c *client) read() (api.Requester, error) {
	// log.Printf("ReadRLock	%d	%#v", c.userID, *req)
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var req api.Requester
	dec := gob.NewDecoder(c.reader)
	err := dec.Decode(&req)

	return req, err
}

type handler func(c *client, req *api.Requester)

func handleIdentify(c *client, req *api.Requester) {
	// log.Printf("Request_Identify	userID:%d", c.userID)

	sd.clientMap.Store(c.userID, c)

	// res := api.Responser(api.IdentifyResponse{UserID: c.userID})
	res := api.IdentifyResponse{UserID: c.userID}
	err := c.send(res)
	if err != nil {
		log.Printf("err	userID:%d	err:%v", c.userID, err)
	}

	// log.Printf("Response_Identify	userID:%d	res:%#v", c.userID, res)
}

func handleList(c *client, req *api.Requester) {
	// log.Printf("Request_List	userID:%d", c.userID)

	var userIDs []uint64
	count := 0
	sd.clientMap.Range(func(k, v interface{}) bool {
		targetUserID := k.(uint64)
		if targetUserID == c.userID {
			return true
		}

		if count >= api.MaxUserListLength {
			return false
		}

		userIDs = append(userIDs, targetUserID)
		count++
		return true
	})

	res := api.ListResponse{UserIDs: userIDs}
	err := c.send(res)
	if err != nil {
		log.Printf("err	userID:%d	err:%v", c.userID, err)
	}
	// log.Printf("Response_List	userID:%d	res:%v", c.userID, res)
}

func handleRelay(c *client, req *api.Requester) {
	// log.Printf("Request_Relay	userID:%d	req:%#v", c.userID, req)
	relayRequest := (*req).(api.RelayRequest)
	res := api.RelayResponse{UserID: c.userID, Message: relayRequest.Message}

	request := (*req).(api.RelayRequest)
	toUserIdsLen := len(request.ToUserIDs)
	if toUserIdsLen > api.MaxUserListLength {
		log.Printf("RELAY_ERR	OVER_TO_USER_IDS_LENGTH	userID:%d	len:%d", c.userID, toUserIdsLen)
		return
	}

	go func() {
		relayMessages <- &relayRequest
	}()

	err := c.send(res)
	if err != nil {
		log.Printf("err	userID:%d	err:%v", c.userID, err)
	}
	// log.Printf("Response_Relay	userID:%dd	res:%v", c.userID, res)
}

func runServer(ctx context.Context, network string, address string) error {

	sd = new(sharedData)

	l, err := net.Listen(network, address)
	if err != nil {
		log.Println(l, err)
		return err
	}
	defer l.Close()
	log.Printf("StartServer	%s:%s", network, address)

	go func() {
		for {
			select {
			case <-ctx.Done():
				l.Close()
				return
			}
		}
	}()

	var wg sync.WaitGroup

	for {

		select {
		case <-ctx.Done():
			wg.Wait()
			log.Printf("Done_runServer")
			return ctx.Err()
		default:

		}

		conn, err := l.Accept()
		if err != nil {
			log.Printf("listen.Accept	err:%v", err)
			continue
		}

		err = conn.(*net.TCPConn).SetKeepAlive(true)
		if err != nil {
			log.Printf("ERR_SET_KEEPALIVE	%v", err)
			continue
		}

		err = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAliveSeconds * time.Second)
		if err != nil {
			log.Printf("ERR_SET_KEEPALIVE_PERIOD	%v", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			handleConnection(ctx, conn)
		}()
	}

}

func handleConnection(ctx context.Context, conn net.Conn) {

	defer conn.Close()

	errs := make(chan error)

	userID := atomic.AddUint64(&sd.currentUserID, 1)
	log.Printf("Connected	userID:%d", userID)

	c := &client{
		userID: userID,
		conn:   conn,
		writer: bufio.NewWriter(conn),
		reader: bufio.NewReader(conn),
	}

	go func() {
		for {
			req, err := c.read()
			if err != nil {
				errs <- err
				return
			}

			// log.Printf("Request	userID:%d	req:%#v", userID, req)

			handler, ok := handleMap[req.GetRequestType()]
			if ok {
				handler(c, &req)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			c.conn.Close()
		case err := <-errs:
			log.Printf("ConnectionClosed	userID:%d	%v", userID, err)
			sd.clientMap.Delete(userID)
			return
		}
	}
}
