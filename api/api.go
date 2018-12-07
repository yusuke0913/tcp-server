package api

import (
	"bytes"
	"encoding/gob"
	"log"
)

const (
	MaxRelayMessageLength = 1024 * 1024

	RequestTypeIdentify uint8 = 1
	RequestTypeList     uint8 = 2
	RequestTypeRelay    uint8 = 3

	ResponseTypeIdentify  uint8 = 1
	ResponseTypeList      uint8 = 2
	ResponseTypeRelay     uint8 = 3
	ResponseTypeRelayDone uint8 = 4

	MaxUserListLength = 255
)

type Requester interface {
	Println()
	GetRequestType() uint8
}

type Responser interface {
	Println()
	GetResponseType() uint8
}

type Message interface {
	Println()
}

type IdentifyRequest struct {
}

func (r IdentifyRequest) Println() {
	log.Printf("IdentifyRequest	%#v", r)
}

func (r IdentifyRequest) GetRequestType() uint8 {
	return RequestTypeIdentify
}

type IdentifyResponse struct {
	UserID uint64
}

func (r IdentifyResponse) Println() {
	log.Printf("IdentifyResponse	%#v", r)
}

func (r IdentifyResponse) GetResponseType() uint8 {
	return ResponseTypeIdentify
}

type ListRequest struct {
	UserID uint64
}

func (r ListRequest) Println() {
	log.Printf("ListRequest	%#v", r)
}

func (r ListRequest) GetRequestType() uint8 {
	return RequestTypeList
}

type ListResponse struct {
	UserIDs []uint64
}

func (r ListResponse) Println() {
	log.Printf("ListResponse:	%v", r.UserIDs)
}

func (r ListResponse) GetResponseType() uint8 {
	return ResponseTypeList
}

type RelayRequest struct {
	UserID    uint64
	Message   string
	ToUserIDs []uint64
}

func (r RelayRequest) Println() {
	log.Printf("RelayRequest	%#v", r)
}

func (r RelayRequest) GetRequestType() uint8 {
	return RequestTypeRelay
}

type RelayResponse struct {
	UserID  uint64
	Message string
}

func (r RelayResponse) Println() {
	log.Println("RelayResponse	%#v", r)
}

func (r RelayResponse) GetResponseType() uint8 {
	return ResponseTypeRelay
}

type RelayDoneResponse struct {
	Count uint8
}

func (r RelayDoneResponse) Println() {
	log.Println("RelayDoneResponse	%#v", r)
}

func (r RelayDoneResponse) GetResponseType() uint8 {
	return ResponseTypeRelayDone
}

func Register() {
	gob.Register(IdentifyRequest{})
	gob.Register(IdentifyResponse{})
	gob.Register(ListRequest{})
	gob.Register(ListResponse{})
	gob.Register(RelayRequest{})
	gob.Register(RelayResponse{})
	gob.Register(RelayDoneResponse{})
}

func Encode(m Message) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(&m)
	if err != nil {
		log.Fatalf("%#v", err)
		return nil, err
	}
	return buf.Bytes(), nil
}
