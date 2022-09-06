package serializer

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtobufToJSON converts protocol buffer message to JSON string
func ProtobufToJSON(message proto.Message) ([]byte, error) {
	marshaler := protojson.MarshalOptions{
		Indent:          "	",
		UseEnumNumbers:  false,
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}
	data, err := marshaler.Marshal(message)
	return data, err
}

// JSONToProtobufMessage converts JSON string to protocol buffer message
func JSONToProtobufMessage(data []byte, message proto.Message) error {
	return protojson.Unmarshal(data, message)
}
