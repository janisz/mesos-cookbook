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

func main() {
	user := "root"
	name := "simple_framework"
	subscribe(FrameworkInfo{
		User: &user,
		Name: &name,
	})
}

func subscribe(frameworkInfo FrameworkInfo) error {
	body, err := json.Marshal(struct {
		Type      string         `json:"type"`
		Subscribe Call_Subscribe `json:"subscribe"`
	}{
		Type:      Call_SUBSCRIBE.String(),
		Subscribe: Call_Subscribe{FrameworkInfo: &frameworkInfo},
	})

	if err != nil {
		log.Printf("%v Error: %v", log.Llongfile, err)
		return err
	}
	res, err := http.Post(master+path, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		log.Printf("%v Error: %v", log.Llongfile, err)
		return err
	}
	defer res.Body.Close()

	reader := bufio.NewReader(res.Body)
	line, err := reader.ReadString('\n')
	bytesCount, err := strconv.Atoi(strings.Trim(line, "\n"))
	if err != nil {
		log.Printf("%v Error: %v", log.Llongfile, err)
		return err
	}
	for {
		line, err = reader.ReadString('\n')
		line = strings.Trim(line, "\n")
		data := line[:bytesCount]
		bytesCount, err = strconv.Atoi((line[bytesCount:]))

		if err != nil {
			log.Printf("%v Error: %v", log.Llongfile, err)
			return err
		}
		var sub Event
		json.Unmarshal([]byte(data), &sub)
		log.Printf("Got: [%s]", sub.String())
	}
}
