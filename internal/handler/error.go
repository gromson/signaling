package handler

import "encoding/json"

const (
	errorCodePairing = 100 + iota
	errorCodeCall
	errorCodePairID
)

type messageHandleError struct {
	Code int    `json:"code"`
	Desc string `json:"desc"`
}

func (e *messageHandleError) Encode() ([]byte, error) {
	return json.Marshal(e)
}
