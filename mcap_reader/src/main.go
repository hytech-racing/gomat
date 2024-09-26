package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jhump/protoreflect/dynamic"

	"raw-matlab-converter/utils"

	"github.com/foxglove/mcap/go/mcap"
)

type Parser struct {
	mcapUtils      *utils.McapUtils
	allSignalData  map[string]map[string]interface{}
	failedMessages [][2]interface{}
	firstTime      *float64
}

func main() {
	argsWithoutProg := os.Args[1:]

	mcapFilePath := argsWithoutProg[0]
	file, err := os.Open(mcapFilePath)
	if err != nil {
		log.Fatalf("error opening file %v: %v", mcapFilePath, err)
	}

	// utils copied over from data_acq_cloud
	mcapUtils := utils.NewMcapUtils()

	reader, err := mcapUtils.NewReader(file)
	if err != nil {
		log.Fatalf("could not create mcap reader: %v", err)
	}

	info, err := reader.Info()
	if err != nil {
		log.Fatalf("could not get mcap info: %v", err)
	}

	message_iterator, err := reader.Messages()
	if err != nil {
		log.Fatalf("could not get mcap mesages: %v", err)
	}

	parser := Parser{
		mcapUtils:      utils.NewMcapUtils(),
		allSignalData:  make(map[string]map[string]interface{}),
		failedMessages: make([][2]interface{}, 0),
		firstTime:      nil,
	}

	parser.mcapUtils.LoadAllSchemas(info)
	for {
		schema, channel, message, err := message_iterator.NextInto(nil)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			log.Fatalf("error reading mcap message: %v", err)
		}

		if schema == nil {
			log.Printf("no schema found for channel ID: %d, channel: %s", message.ChannelID, channel)
			continue
		}

		newFailedMessage := make([][2]interface{}, 0)

		for _, failedMessage := range parser.failedMessages {
			if schema.Name == "hytech_msgs.MCUCommandData" {
				// fmt.Println("")
			}
			err = parser.processMessage(failedMessage[0].(*mcap.Message), failedMessage[1].(*mcap.Schema))
			if err != nil {
				newFailedMessage = append(newFailedMessage, [2]interface{}{failedMessage[0], failedMessage[1]})
			}
		}

		err = parser.processMessage(message, schema)
		if err != nil {
			newFailedMessage = append(newFailedMessage, [2]interface{}{message, schema})
		}

		parser.failedMessages = newFailedMessage
	}

	if len(parser.failedMessages) != 0 {
		fmt.Errorf("could not finish decoding all messages")
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(parser.allSignalData)
	if err != nil {
		fmt.Println("Error serializing data:", err)
		os.Exit(1)
	}
	cmd := exec.Command("matlab_writer.py", "-p", file.Name())
	cmd.Stdin = bytes.NewReader([]byte(jsonData))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to execute Python script: %s", err)
	}

	fmt.Println(string(output))
}

func (p *Parser) processMessage(message *mcap.Message, schema *mcap.Schema) error {
	decodedMessage, err := p.mcapUtils.GetDecodedMessage(schema, message)
	if err != nil {
		p.failedMessages = append(p.failedMessages, [2]interface{}{message, schema})
		return fmt.Errorf("could not decode message: %v", err)
	}

	if decodedMessage == nil || decodedMessage.Data == nil {
		return fmt.Errorf("could not decode message: %v", decodedMessage)
	}
	signalValues := decodedMessage.Data

	// Some topics could be in the format of "hytech_msgs.MCUOutputData", but it's cleaner to just have MCUOutputData
	trimmedTopicSlice := strings.Split(decodedMessage.Topic, ".")
	trimmedTopic := trimmedTopicSlice[len(trimmedTopicSlice)-1]

	if p.firstTime == nil {
		firstValue := float64(decodedMessage.LogTime) / 1e9
		p.firstTime = &firstValue
	}

	if p.allSignalData[trimmedTopic] == nil {
		p.allSignalData[trimmedTopic] = make(map[string]interface{})
	}

	for signalName, value := range signalValues {
		p.processSignalValue(trimmedTopic, signalName, value, float64(decodedMessage.LogTime)/1e9)
	}

	return nil
}

func (p *Parser) processSignalValue(topic, signalName string, value interface{}, logTime float64) {
	dynamicMessage, ok := value.(*dynamic.Message)
	if ok {
		if p.allSignalData[topic][signalName] == nil {
			p.allSignalData[topic][signalName] = make(map[string]interface{})
		}

		if dynamicMessage == nil {
			return
		}

		// Process nested dynamic message fields
		currentNest := p.allSignalData[topic][signalName]
		p.addNestedValues(currentNest.(map[string]interface{}), dynamicMessage, logTime)

	} else {
		// Non-dynamic message values are processed normally
		if p.allSignalData[topic][signalName] == nil {
			p.allSignalData[topic][signalName] = make([][]float64, 0)
		}

		p.allSignalData[topic][signalName] = append(p.allSignalData[topic][signalName].([][]float64), []float64{logTime - *p.firstTime, getFloatValueOfInterface(value)})
	}
}

// Function to add nested values from dynamic message fields recursively
func (p *Parser) addNestedValues(nestedMap map[string]interface{}, dynamicMessage *dynamic.Message, logTime float64) {
	fieldNames := dynamicMessage.GetKnownFields()
	// Get all the field descriptors associated with this message
	for _, field := range fieldNames {
		fieldName := field.GetName()

		// Each dynamic message has field descriptors, not data. We need to extract those field descriptors and then use them
		// to figure out what data values are in there. The value could be another map, a list of values, or just a single value.
		// NOTE: We don't need to figure out what descriptors there are  because we already decoded these messages in the GetDecodedMessage logic,
		// so they are just know associated with these dynamic.Message's.
		fieldDescriptor := dynamicMessage.FindFieldDescriptorByName(fieldName)
		if fieldDescriptor == nil {
			continue
		}

		decodedValue := dynamicMessage.GetField(fieldDescriptor)
		if decodedValue == nil {
			continue
		}

		unboxedNested, isNestedMessage := decodedValue.(*dynamic.Message)
		if isNestedMessage {
			// If it is a nested message and another map doesn't exist for it, we will make one to use.
			if _, ok := nestedMap[fieldName]; !ok {
				nestedMap[fieldName] = make(map[string]interface{})
			}

			p.addNestedValues(nestedMap[fieldName].(map[string]interface{}), unboxedNested, logTime)
		} else {
			if _, ok := nestedMap[fieldName]; !ok {
				nestedMap[fieldName] = make([][]float64, 0)
			}

			nestedMap[fieldName] = append(nestedMap[fieldName].([][]float64), []float64{logTime - *p.firstTime, getFloatValueOfInterface(decodedValue)})
		}
	}
}

func getFloatValueOfInterface(val interface{}) float64 {
	if val == nil {
		return 0
	}

	var out float64

	switch x := val.(type) {
	case int32:
		out = float64(x)
	case uint64:
		out = float64(x)
	case float32:
		out = float64(x)
	case string:
		i, err := strconv.Atoi(x)
		if err != nil {
			panic(err)
		}
		out = float64(i)
	case bool:
		if val.(bool) {
			out = 1
		} else {
			out = 0
		}
	}

	return out
}
