package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	var request map[string]interface{}

	destination := os.Args[1]

	{
		err := json.NewDecoder(os.Stdin).Decode(&request)
		if err != nil {
			fatal("parsing request", err)
		}
	}

	response := make(map[string]interface{})
	response["version"] = request["version"]

	// Extract timestamp from version which may be a string or a map[string]interface{}
	var timestamp string
	if v, ok := request["version"]; ok {
		switch vv := v.(type) {
		case string:
			timestamp = vv
		case map[string]interface{}:
			if ts, ok := vv["timestamp"].(string); ok {
				timestamp = ts
			}
		}
	}
	if timestamp == "" {
		fatal("extracting version timestamp", fmt.Errorf("unexpected version format: %T", request["version"]))
	}

	{
		err := ioutil.WriteFile(filepath.Join(destination, "timestamp"), []byte(timestamp), 0644)
		if err != nil {
			fatal("writing timestamp file", err)
		}
	}

	{
		err := json.NewEncoder(os.Stdout).Encode(&response)
		if err != nil {
			fatal("serializing response", err)
		}
	}
}

func fatal(doing string, err error) {
	fmt.Fprintf(os.Stderr, "Error "+doing+": "+err.Error()+"\n")
	os.Exit(1)
}
