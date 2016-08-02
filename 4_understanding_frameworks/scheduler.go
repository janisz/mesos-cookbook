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
	"io"
	"fmt"
)

const master = "http://10.10.10.10:5050"
const path = "/api/v1/scheduler"

var marshaller = jsonpb.Marshaler{
	EnumsAsInts: false,
	Indent: "  ",
	OrigName: true,
}
var mesosStreamId string

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
		case Event_SUBSCRIBED:
			log.Print("Subscribed")
			frameworkInfo.Id = sub.Subscribed.FrameworkId
			mesosStreamId = res.Header.Get("Mesos-Stream-Id")
		case Event_HEARTBEAT: log.Print("PING")
		case Event_OFFERS: {

			offerIds := []*OfferID{}
			for _, offer := range sub.Offers.Offers {
				offerIds = append(offerIds, offer.Id)
			}

			decline := &Call{
				Type: Call_DECLINE.Enum(),
				FrameworkId: frameworkInfo.Id,
				Decline: &Call_Decline{OfferIds: offerIds},
			}
			body, err := marshaller.MarshalToString(decline)
			if err != nil {
				log.Printf("%v Error: %v", log.Llongfile, err)
				return err
			}
			log.Print(body)
			req, err := http.NewRequest("POST", master + path, bytes.NewBuffer([]byte(body)))
			if (err != nil) {
				return err
			}
			req.Header.Set("Mesos-Stream-Id", mesosStreamId)
			req.Header.Set("Content-Type", "application/json")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("%v Error: %v", log.Llongfile, err)
				return err
			}
			if (res.StatusCode != 202) {
				io.Copy(os.Stderr, res.Body)
				return fmt.Errorf("%s", "Error")
			}
			res.Body.Close()
		}
		}

	}
}
