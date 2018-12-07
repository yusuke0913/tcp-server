package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/yusuke0913/tcp-server/api"
)

const (
	timeoutDuration        = 3 * time.Second
	testConcurrency        = 24
	allTestTimeoutDuration = 10 * time.Second
)

func init() {
	api.Register()
}
func TestNewClient(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx, network, address)
	if err != nil {
		t.Fatalf("Faild NewClient	%#v", err)
	}
}
func TestSend(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, allTestTimeoutDuration)
	// ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error)
	resultChan := make(chan bool)

	for i := 0; i < testConcurrency; i++ {
		go func(number int) {
			result, err := Send(ctx, t, number)
			resultChan <- result
			if err != nil {
				errChan <- err
			}
		}(i)
		log.Printf("Exe	Test	i:%d", i)
	}

	doneNum := 0
	totalResult := true
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			log.Printf("Done	TestSend:%v", err)
			if err != nil {
				t.Fatalf("Failed Result	%v", err)
			}
			return
		case result := <-resultChan:
			if !result {
				totalResult = false
			}

			doneNum++
			if doneNum < testConcurrency {
				break
			}

			if !totalResult {
				t.Fatalf("Failed	Result	%v", result)
			}

			return
		case err := <-errChan:
			t.Fatalf("Error	%v", err)
			return
		}
	}
}

func Send(ctx context.Context, t *testing.T, number int) (bool, error) {
	client, err := NewClient(ctx, network, address)
	if err != nil {
		errMsg := fmt.Sprintf("Faild NewClient	%d	%#v", number, err)
		return false, errors.New(errMsg)
	}
	defer client.conn.Close()

	{
		name := fmt.Sprintf("Identify_%d", number)
		result := t.Run(name, func(t *testing.T) {
			SendIdentify(ctx, t, client)
		})
		log.Printf("result	identify	userID:%d	%v", client.userID, result)
		if !result {
			return false, nil
		}
	}

	{
		name := fmt.Sprintf("List_%d", number)
		result := t.Run(name, func(t *testing.T) {
			SendList(ctx, t, client)
		})
		log.Printf("result	List	userID:%d	%v", client.userID, result)
		if !result {
			return false, nil
		}
	}

	{
		name := fmt.Sprintf("Relay_%d", number)
		result := t.Run(name, func(t *testing.T) {
			SendRelay(ctx, t, client)
		})
		log.Printf("result	relay	userID:%d	%v", client.userID, result)
		if !result {
			return false, nil
		}
	}

	return true, nil
}

func SendIdentify(ctx context.Context, t *testing.T, c *Client) {
	err := c.sendIdentify()
	if err != nil {
		t.Fatalf("Failed SendList	%#v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if c.userID != 0 {
					cancel()
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			if c.userID == 0 {
				t.Fatal("Faild Server didn't answer userID")
			}
			log.Printf("Done	TestSendIdentify	userID:%d", c.userID)
			return
		}
	}
}

func SendList(ctx context.Context, t *testing.T, c *Client) {
	log.Printf("SendList_0	userId:%d", c.userID)
	err := c.sendList()
	if err != nil {
		t.Fatalf("Faild SendList	%#v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	done := false

	log.Printf("SendList_1	userId:%d", c.userID)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("SendList_Done	userId:%d, err:%v", c.userID, ctx.Err())
				return
			default:

				log.Printf("SendList_2	userId:%d", c.userID)

				toUserIdsLength := len(c.toUserIDs)
				if toUserIdsLength <= 0 {
					continue
				}

				log.Printf("Test	toUserIDs	length:%d	%#v", toUserIdsLength, c.toUserIDs)

				if toUserIdsLength > api.MaxRelayMessageLength {
					t.Fatalf("Faild	List	Length	Over	%#v", err)
				}

				for _, userID := range c.toUserIDs {
					if userID == c.userID {
						t.Fatal("Faild	userIds List contains my userID")
					}
				}

				done = true
				cancel()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			if !done {
				t.Fatalf("Faild	TestSendList	userID:%d", c.userID)
			}
			log.Printf("Done	TestSendList	userID:%d", c.userID)
			return
		}
	}

}

func SendRelay(ctx context.Context, t *testing.T, c *Client) {

	err := c.sendRelay()
	if err != nil {
		t.Fatalf("Faild SendList	%#v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	done := false
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if c.relaydCount > 0 {
					log.Printf("Done	userID:%d	RelaydCount:%d", c.userID, c.relaydCount)
					done = true
					cancel()
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			if !done {
				t.Fatalf("Faild	TestSendRelay	userID:%d", c.userID)
			}
			log.Printf("Done	TestSendRelay	userID:%d", c.userID)
			return
		}
	}

}
