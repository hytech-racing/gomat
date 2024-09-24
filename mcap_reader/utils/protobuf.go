package utils

import (
	"fmt"

	"github.com/foxglove/mcap/go/mcap"
	"github.com/jhump/protoreflect/desc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// MCAP doesn't have a native protobuf wrapper for golang, so we have to
// dynamically decode the schemas and messages

type ProtobufUtils struct {
	protoDescriptions  map[string]*desc.FileDescriptor
	protoDescriptorSet descriptorpb.FileDescriptorSet
}

type ProtobufMessage struct {
	name string
}

func NewProtobufUtils() *ProtobufUtils {
	return &ProtobufUtils{
		protoDescriptions:  make(map[string]*desc.FileDescriptor),
		protoDescriptorSet: descriptorpb.FileDescriptorSet{},
	}
}

func (pb *ProtobufUtils) loadSchema(schema *mcap.Schema) (*desc.FileDescriptor, error) {
	fdSet := &pb.protoDescriptorSet
	if err := proto.Unmarshal(schema.Data, fdSet); err != nil {
		return nil, fmt.Errorf("failed to parse schema data: %w", err)
	}

	files := make([]*desc.FileDescriptor, len(fdSet.GetFile()))
	for i, fd := range fdSet.GetFile() {
		file, err := desc.CreateFileDescriptor(fd)
		if err != nil {
			return nil, fmt.Errorf("failed to create file descriptor for %s: %w", fd.GetName(), err)
		}
		files[i] = file
	}

	pb.protoDescriptions[schema.Name] = files[0]

	return files[0], nil
}

func (pb *ProtobufUtils) GetDecodedSchema(schema *mcap.Schema) (*desc.FileDescriptor, error) {
	i, ok := pb.protoDescriptions[schema.Name]
	if ok {
		return i, nil
	}

	return pb.loadSchema(schema)
}
