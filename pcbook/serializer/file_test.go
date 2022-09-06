package serializer_test

import (
	"testing"

	"github.com/This-Is-Prince/pcbook/pb"
	"github.com/This-Is-Prince/pcbook/sample"
	"github.com/This-Is-Prince/pcbook/serializer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestFileSerializer(t *testing.T) {
	t.Parallel()

	binaryFile := "../tmp/laptop.bin"
	laptop1 := sample.NewLaptop()
	err := serializer.WriteProtobufToBinaryFile(laptop1, binaryFile)
	require.NoError(t, err)

	laptop2 := &pb.Laptop{}
	err = serializer.ReadProtoBufFromBinaryFile(binaryFile, laptop2)
	require.NoError(t, err)

	require.True(t, proto.Equal(laptop1, laptop2))

	jsonFile := "../tmp/laptop.json"
	laptopJSON1 := sample.NewLaptop()
	err = serializer.WriteProtobufToJSONFile(laptopJSON1, jsonFile)
	require.NoError(t, err)

	laptopJSON2 := &pb.Laptop{}
	err = serializer.ReadProtoBufFromJSONFile(jsonFile, laptopJSON2)
	require.NoError(t, err)

	require.True(t, proto.Equal(laptopJSON1, laptopJSON2))
}
