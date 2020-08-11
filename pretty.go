package main

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

func PrettyPrintJSON(pb proto.Message) string {
	m := jsonpb.Marshaler{OrigName: true, Indent: "  ", EmitDefaults: true}
	s, err := m.MarshalToString(pb)
	if err != nil {
		panic(err)
	}
	return s
}
