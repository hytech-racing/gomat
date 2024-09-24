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

	"raw-matlab-converter/utils"

	"github.com/foxglove/mcap/go/mcap"
)

type Parser struct {
	mcapUtils      *utils.McapUtils
	allSignalData  map[string]map[string][][]interface{}
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

	message_iterator, err := reader.Messages()
	if err != nil {
		log.Fatalf("could not get mcap mesages: %v", err)
	}

	parser := Parser{
		mcapUtils:      utils.NewMcapUtils(),
		allSignalData:  make(map[string]map[string][][]interface{}),
		failedMessages: make([][2]interface{}, 0),
		firstTime:      nil,
	}

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
			err = parser.processMessage(failedMessage[0].(*mcap.Message), failedMessage[1].(*mcap.Schema))
			if err != nil {
				newFailedMessage = append(newFailedMessage, [2]interface{}{failedMessage[0], failedMessage[1]})
			}
		}

		err = parser.processMessage(message, schema)
		if err != nil {
			parser.failedMessages = append(parser.failedMessages, [2]interface{}{message, schema})
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

	signalValues := decodedMessage.Data

	if p.firstTime == nil {
		firstValue := float64(decodedMessage.LogTime) / 1e9
		p.firstTime = &firstValue
	}

	if p.allSignalData[decodedMessage.Topic] == nil {
		p.allSignalData[decodedMessage.Topic] = make(map[string][][]interface{})
	}

	for signalName, value := range signalValues {
		floatValue := getFloatValueOfInterface(value)
		signalSlice := p.allSignalData[decodedMessage.Topic][signalName] // All signal data for one signal value.

		timeSec := (float64(decodedMessage.LogTime) / 1e9) - *p.firstTime
		singleRowToAdd := []interface{}{timeSec, floatValue}

		signalSlice = append(signalSlice, singleRowToAdd)
		p.allSignalData[decodedMessage.Topic][signalName] = signalSlice
	}

	return nil
}

func getFloatValueOfInterface(val interface{}) float64 {
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
