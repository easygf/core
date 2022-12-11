package utils

import (
	"github.com/easygf/core/log"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

func Pb2JsonSkipDefaults(msg proto.Message) (string, error) {
	var m = jsonpb.Marshaler{
		EmitDefaults: false,
		OrigName:     true,
	}
	j, err := m.MarshalToString(msg)
	if err != nil {
		log.Error("proto MarshalToString err:", err)
		return "", err
	}
	return j, nil
}
