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
)

func main() {
	argsWithoutProg := os.Args[1:]

	var firstTime *float64 = nil
	mcapFilePath := argsWithoutProg[0]
	file, err := os.Open(mcapFilePath)
	if err != nil {
		log.Fatalf("error opening file %v: %v", mcapFilePath, err)
	}

	// utils copied over from data_acq_cloud
	mcapUtils := utils.NewMcapUtils()

	allSignalData := make(map[string]map[string][][]interface{})

	reader, err := mcapUtils.NewReader(file)
	if err != nil {
		log.Fatalf("could not create mcap reader")
	}

	message_iterator, err := reader.Messages()
	if err != nil {
		log.Fatalf("could not get mcap mesages")
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

		decodedMessage, err := mcapUtils.GetDecodedMessage(schema, message)
		if err != nil {
			log.Fatalf("could not decode message: %v", err)
		}

		signalValues := decodedMessage.Data

		if firstTime == nil {
			firstValue := float64(decodedMessage.LogTime) / 1e9
			firstTime = &firstValue
		}

		if allSignalData[decodedMessage.Topic] == nil {
			allSignalData[decodedMessage.Topic] = make(map[string][][]interface{})
		}

		for signalName, value := range signalValues {
			floatValue := getFloatValueOfInterface(value)
			signalSlice := allSignalData[decodedMessage.Topic][signalName] // All signal data for one signal value.

			timeSec := (float64(decodedMessage.LogTime) / 1e9) - *firstTime
			singleRowToAdd := []interface{}{timeSec, floatValue}

			signalSlice = append(signalSlice, singleRowToAdd)
			allSignalData[decodedMessage.Topic][signalName] = signalSlice
		}
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(allSignalData)
	if err != nil {
		fmt.Println("Error serializing data:", err)
		os.Exit(1)
	}
	cmd := exec.Command("matlab_writer.py")
	cmd.Stdin = bytes.NewReader([]byte(jsonData))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to execute Python script: %s", err)
	}

	fmt.Println(string(output))
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
