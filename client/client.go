package main

import (
	"bufio"
	"context"
	"encoding/gob"
	"log"
	"net"
	"sync"
	"time"

	"github.com/yusuke0913/tcp-server/api"
)

type Client struct {
	conn net.Conn

	mutex  sync.RWMutex
	reader *bufio.Reader

	wm     sync.Mutex
	writer *bufio.Writer

	userID      uint64
	toUserIDs   []uint64
	relaydCount uint8
}

func NewClient(ctx context.Context, network string, address string) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAliveSeconds * time.Second)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	client := &Client{
		conn:   conn,
		userID: 0,
		writer: bufio.NewWriter(conn),
		reader: bufio.NewReader(conn),
	}

	// start receive handler
	go func() {
		client.receiveHandler(ctx)
	}()

	// register user id
	err = client.sendIdentify()
	if err != nil {
		return nil, err
	}

	// wait for getting user id from server
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Millisecond * 100)
				if client.userID != 0 {
					log.Printf("IDENTIFIED_USER_ID	userID:%d", client.userID)
					return
				}
				log.Printf("Waiting_Server_Response	userID:%d", client.userID)
			}
		}
	}()

	// send randomly
	go func() {
		client.sendRandomly(ctx)
	}()

	return client, nil
}

func (c *Client) send(req api.Requester) error {
	// log.Printf("SendLock	userID:%d	%#v", c.userID, *req)
	c.wm.Lock()
	defer c.wm.Unlock()

	var (
		err error
	)

	bytes, err := api.Encode(req)
	if err != nil {
		log.Printf("ERR_SEND_ENCODE	userID:%d	err:%v	reqType:%v	req:%v", c.userID, err, req.GetRequestType(), req)
		return err
	}

	_, err = c.writer.Write(bytes)
	if err != nil {
		log.Printf("ERR_WRITER_WRITE	userID:%d	err:%v	reqType:%v	req:%v", c.userID, err, req.GetRequestType(), req)
		return err
	}

	err = c.writer.Flush()
	if err != nil {
		log.Printf("ERR_WRITER_FLUSH	userID:%d	err:%v	reqType:%v	req:%v", c.userID, err, req.GetRequestType(), req)
		return err
	}

	// log.Printf("Send	userID:%d	bytes:%d	req:%#v", c.userID, n, *req)
	return err
}

func (c *Client) read(res *api.Responser) error {
	// log.Printf("ReadRLock	%d	%#v", c.userID, *res)
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	dec := gob.NewDecoder(c.reader)
	err := dec.Decode(&res)
	return err
}

func (c *Client) receiveHandler(ctx context.Context) {
	errs := make(chan error)

	go func() {
		for {
			var res api.Responser
			err := c.read(&res)
			if err != nil {
				errs <- err
				continue
			}

			// log.Printf("Response	userID:%d	req:%#v", c.userID, res)
			handler, ok := handleMap[res.GetResponseType()]
			if !ok {
				errs <- err
				continue
			}
			handler(c, &res)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Printf("CLOSED_CONNECTION	userID:%d", c.userID)
			c.conn.Close()
		case err := <-errs:
			log.Printf("ERR_RECEIVE_HANDLER	userID:%d	%v", c.userID, err)
			return
		}
	}

}

func (c *Client) sendRandomly(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var (
				err error
			)
			time.Sleep(time.Millisecond * 10)
			err = c.sendIdentify()
			if err != nil {
				return
			}
			time.Sleep(time.Millisecond * 10)
			err = c.sendList()
			if err != nil {
				return
			}
			time.Sleep(time.Millisecond * 10)
			err = c.sendRelay()
			if err != nil {
				return
			}
		}
	}
}
func (c *Client) sendIdentify() error {
	req := api.IdentifyRequest{}
	return c.send(req)
}

func (c *Client) sendList() error {
	req := api.ListRequest{UserID: c.userID}
	return c.send(req)
}

func (c *Client) sendRelay() error {
	// request
	lenToUserIDs := len(c.toUserIDs)
	if lenToUserIDs <= 0 {
		// log.Printf("requestRelay	ERR	toUserIdsLength:0")
		return nil
	}

	if lenToUserIDs > api.MaxUserListLength {
		log.Printf("requestRelay	ERR_TOO_MANY_TO_USER_IDS	userIDs:%d	len:%d", c.userID, lenToUserIDs)
		return nil
	}

	message := "Hello from the other side"

	req := api.RelayRequest{
		UserID:    c.userID,
		ToUserIDs: c.toUserIDs,
		Message:   message,
	}

	return c.send(req)
}

type handler func(c *Client, res *api.Responser)

func handleIdentify(c *Client, res *api.Responser) {
	c.userID = (*res).(api.IdentifyResponse).UserID
	// log.Printf("Response	Identify	userID:%d	res:%#v", c.userID, (*res).(api.IdentifyResponse))
}

func handleList(c *Client, res *api.Responser) {
	// log.Printf("Response_List	userID:%d	res:%#v", c.userID, (*res).(api.ListResponse))
	listRes := (*res).(api.ListResponse)
	if len(listRes.UserIDs) > api.MaxRelayMessageLength {
		return
	}
	c.toUserIDs = listRes.UserIDs
}

func handleRelay(c *Client, res *api.Responser) {
	// log.Printf("Response_Relay	userID:%d	res:%#v", c.userID, (*res).(api.RelayResponse))
}

func handleRelayDone(c *Client, res *api.Responser) {
	// log.Printf("Response_RelayDone	userID:%d	res:%#v", c.userID, (*res).(api.RelayDoneResponse))
	c.relaydCount = (*res).(api.RelayDoneResponse).Count
}
