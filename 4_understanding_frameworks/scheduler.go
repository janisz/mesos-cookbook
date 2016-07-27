package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const master = "http://10.10.10.10:5050"
const path = "/api/v1/scheduler"

type baseEvent struct {
	Type string `json:"type"`
}

type subscribedEvent struct {
	Type       string `json:"type"`
	Subscribed string `json:"subscribed"`
}

type subscribed struct {
	FrameworkID string `json:"framework_id"`
}

type frameworkID struct {
	Value string `json:"value"`
}

func getEventType(jsonBlob []byte) (string, error) {
	event := baseEvent{}
	err := json.Unmarshal(jsonBlob, &event)
	if err != nil {
		return "", err
	} else if event.Type == "" {
		return "", nil
	} else {
		return event.Type, nil
	}
}

func main() {
	subscribe()
}

func subscribe() error {
	subscribeBody := `{
		"type": "SUBSCRIBE",
		"subscribe": {
		"framework_info": {
			"user" :  "root",
			"name" :  "simple_framework"
		},
		"force" : true
		}
	}`

	res, err := http.Post(master+path, "application/json", bytes.NewBuffer([]byte(subscribeBody)))
	if err != nil {
		log.Print(err)
		return err
	}
	defer res.Body.Close()

	reader := bufio.NewReader(res.Body)
	line, err := reader.ReadString('\n')
	bytesCount, err := strconv.Atoi(strings.Trim(line, "\n"))
	if err != nil {
		log.Print(err)
		return err
	}
	for {
		line, err = reader.ReadString('\n')
		line = strings.Trim(line, "\n")
		data := line[:bytesCount]
		bytesCount, err = strconv.Atoi((line[bytesCount:]))

		if err != nil {
			log.Print(err)
			return err
		}
		eventType, err := getEventType([]byte(data))
		if err != nil {
			log.Print(err)
			return err
		}

		switch eventType {
		case "ERROR":
			log.Printf("Type %s [%s]", eventType, line)
		case "OFFERS":
			log.Printf("Type %s [%s]", eventType, line)
		case "SUBSCRIBED":
			var sub subscribedEvent
			json.Unmarshal([]byte(data), &sub)
			log.Printf("Type %s [%s]", eventType, sub)
		case "HEARTBEAT":
			log.Printf("Type %s", eventType)
		}
	}
}
