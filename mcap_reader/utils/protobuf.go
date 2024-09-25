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
	protoDescriptorSet *descriptorpb.FileDescriptorSet
}

type ProtobufMessage struct {
	name string
}

func NewProtobufUtils() *ProtobufUtils {
	return &ProtobufUtils{
		protoDescriptions:  make(map[string]*desc.FileDescriptor),
		protoDescriptorSet: &descriptorpb.FileDescriptorSet{},
	}
}

func (pb *ProtobufUtils) loadSchema(schema *mcap.Schema) (*desc.FileDescriptor, error) {
	if schema.Name == "hytech_msgs.MCUCommandData" {
		fmt.Println("")
	}

	fdSet := &pb.protoDescriptorSet
	if err := proto.Unmarshal(schema.Data, *fdSet); err != nil {
		return nil, fmt.Errorf("failed to parse schema data: %w", err)
	}

	//files := make([]*desc.FileDescriptor, len((*fdSet).GetFile()))
	//success := false
	//fileIdx := -1
	//for i, fd := range (*fdSet).GetFile() {
	//	file, err := desc.CreateFileDescriptor(fd)
	//	if err != nil {
	//		continue
	//	}
	//	success = true
	//	files[i] = file
	//	fileIdx = i
	//	break
	//}

	fdSetFiles := (*fdSet).GetFile()
	successfulFiles := make([]*desc.FileDescriptor, 0)
	errFiles := make([]*descriptorpb.FileDescriptorProto, 0, len(fdSetFiles))
	for range len(fdSetFiles) {
		for _, fd := range fdSetFiles {
			if len(successfulFiles) == len(fdSetFiles) {
				break
			}

			if pb.protoDescriptions[fd.GetName()] != nil {
				successfulFiles = append(successfulFiles, pb.protoDescriptions[fd.GetName()])
				continue
			}

			file, err := desc.CreateFileDescriptor(fd, successfulFiles...)
			if err != nil {
				errFiles = append(errFiles, fd)
				continue
			}
			successfulFiles = append(successfulFiles, file)
			pb.protoDescriptions[fd.GetName()] = file

			fdSetFiles = errFiles
		}
	}

	if len(errFiles) != 0 {
		return nil, fmt.Errorf("failed to create file descriptors for %v", errFiles)
	}

	if len(successfulFiles) == 0 {
		return nil, nil
	}

	// To find the highestLevelFile, we need to find the one with the largest number of dependencies
	maxDepLen := -1
	var highestLevelFile *desc.FileDescriptor = nil
	for _, succFile := range successfulFiles {
		depLen := len(succFile.AsFileDescriptorProto().Dependency)
		if depLen > maxDepLen {
			highestLevelFile = succFile
			maxDepLen = depLen
		}
	}

	pb.protoDescriptions[schema.Name] = highestLevelFile
	fdProto := highestLevelFile.AsFileDescriptorProto()
	pb.protoDescriptorSet.File = append(pb.protoDescriptorSet.File, fdProto)

	return highestLevelFile, nil
}

func (pb *ProtobufUtils) GetDecodedSchema(schema *mcap.Schema) (*desc.FileDescriptor, error) {
	i, ok := pb.protoDescriptions[schema.Name]
	if ok {
		return i, nil
	}

	return pb.loadSchema(schema)
}
