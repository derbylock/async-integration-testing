package db

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

type ProtoCodec struct{}

func (c ProtoCodec) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("can't marshal object, it is not a proto message")
	}
	return proto.Marshal(msg)
}

// Unmarshal decodes a gob value into a Go value.
func (c ProtoCodec) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("can't unmarshal object, it is not a proto message")
	}
	proto.Unmarshal(data, msg)
	return nil
}

var PROTO_CODEC = ProtoCodec{}
