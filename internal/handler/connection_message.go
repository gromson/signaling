package handler

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

type MessageType uint8

const (
	// incoming messages
	incomingMessageSignaling = MessageType(iota + 1)
	incomingMessageCall
	incomingMessageAnswer

	// outgoing messages
	outgoingMessageSignaling
	outgoingMessageCallInitialized
	outgoingMessageAnswerAccepted
	outgoingMessageError
)

type connectionMessage struct {
	Typ     MessageType
	Content []byte
}

func newConnectionMessageFromBytes(data []byte) (*connectionMessage, error) {
	r := bytes.NewReader(data)
	var typ MessageType
	if err := binary.Read(r, binary.LittleEndian, &typ); err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "could not parse binary message")
	}

	return &connectionMessage{
		Typ:     typ,
		Content: data[1:],
	}, nil
}

func (m connectionMessage) Encode() []byte {
	return append([]byte{byte(m.Typ)}, m.Content...)
}
