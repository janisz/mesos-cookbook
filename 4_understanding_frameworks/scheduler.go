package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"github.com/golang/protobuf/jsonpb"
	"os"
)

const master = "http://10.10.10.10:5050"
const path = "/api/v1/scheduler"

var marshaller = jsonpb.Marshaler{
	EnumsAsInts: false,
	Indent: "  ",
	OrigName: true,
}

type scheduler struct {

}

func main() {
	user := "root"
	name := "simple_framework"
	hostname, err := os.Hostname()
	if (err != nil) {
		log.Fatal(err)
	}
	log.Fatal(subscribe(FrameworkInfo{
		User: &user,
		Name: &name,
		Hostname: &hostname,
	}))
}

func subscribe(frameworkInfo FrameworkInfo) error {

	subscribeCall := &Call{
		Type: Call_SUBSCRIBE.Enum(),
		Subscribe: &Call_Subscribe{FrameworkInfo: &frameworkInfo},
	}

	body, err := marshaller.MarshalToString(subscribeCall)

	if err != nil {
		log.Printf("%v Error: %v", log.Llongfile, err)
		return err
	}
	log.Print(body)
	res, err := http.Post(master + path, "application/json", bytes.NewBuffer([]byte(body)))
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

		switch *sub.Type {
		case Event_HEARTBEAT: log.Print("PING")
		case Event_OFFERS: log.Print("Offer")
		case Event_OFFERS: log.Print("Offer")
		}

	}
}
