package db

import (
	"google.golang.org/protobuf/proto"
)

type ProtoCodec struct{}

func (c ProtoCodec) Marshal(v proto.Message) ([]byte, error) {
	return proto.Marshal(v)
}

// Unmarshal decodes a gob value into a Go value.
func (c ProtoCodec) Unmarshal(data []byte, v proto.Message) error {
	return proto.Unmarshal(data, v)
}

var PROTO_CODEC = ProtoCodec{}
