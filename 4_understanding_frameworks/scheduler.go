package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"sync/atomic"
)

const master = "http://10.10.10.10:5050"
const path = "/api/v1/scheduler"

var marshaller = jsonpb.Marshaler{
	EnumsAsInts: false,
	Indent:      "  ",
	OrigName:    true,
}
var mesosStreamID string

var frameworkInfo *FrameworkInfo

var commandChan = make(chan string, 100)

var taskId uint64

func main() {
	user := "root"
	name := "simple_framework"
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	frameworkInfo = &FrameworkInfo{
		User:     &user,
		Name:     &name,
		Hostname: &hostname,
	}

	go subscribe()
	http.HandleFunc("/", web)               // set router
	err = http.ListenAndServe(":9090", nil) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func web(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cmd := r.Form["cmd"][0]
	commandChan <- cmd
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Scheduled: %s", cmd)
}

func subscribe() error {

	subscribeCall := &Call{
		Type:      Call_SUBSCRIBE.Enum(),
		Subscribe: &Call_Subscribe{FrameworkInfo: frameworkInfo},
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
			mesosStreamID = res.Header.Get("Mesos-Stream-Id")
		case Event_HEARTBEAT:
			log.Print("PING")
		case Event_OFFERS:
			{

				offerIds := []*OfferID{}
				for _, offer := range sub.Offers.Offers {
					offerIds = append(offerIds, offer.Id)
				}

				select {
				case cmd := <-commandChan:
					firstOffer := sub.Offers.Offers[0]

					TRUE := true
					newTaskId := fmt.Sprint(atomic.AddUint64(&taskId, 1))
					accept := &Call{
						Type: Call_ACCEPT.Enum(),
						Accept: &Call_Accept{
							OfferIds: offerIds,
							Operations: []*Offer_Operation{{
								Type: Offer_Operation_LAUNCH.Enum(),
								Launch: &Offer_Operation_Launch{
									TaskInfos: []*TaskInfo{
										{
											Name: &cmd,
											TaskId: &TaskID{Value: &newTaskId},
											AgentId: firstOffer.AgentId,
											Resources: defaultResources(),
											Command: &CommandInfo{
												Shell: &TRUE,
												Value: &cmd,
											},
										},
									},
								},
							},
							},
						},
					}
					err = call(accept)
					if err != nil {
						return err
					}


				default:
					decline := &Call{
						Type:    Call_DECLINE.Enum(),
						Decline: &Call_Decline{OfferIds: offerIds},
					}
					err = call(decline)
					if err != nil {
						return err
					}
				}
			}
		}

	}
}

func defaultResources() []*Resource {
	CPU := "cpus"
	MEM := "mem"
	cpu := float64(0.1)

	return []*Resource{
		{
			Name: &CPU,
			Type: Value_SCALAR.Enum(),
			Scalar: &Value_Scalar{Value: &cpu},
		},
		{
			Name: &MEM,
			Type: Value_SCALAR.Enum(),
			Scalar: &Value_Scalar{Value: &cpu},
		},
	}

}

// call Sends request to Mesos.
// Returns nil if request was accepted in other case returns error
func call(message *Call) error {
	message.FrameworkId = frameworkInfo.Id
	body, err := marshaller.MarshalToString(message)
	if err != nil {
		log.Printf("%v Error: %v", log.Llongfile, err)
		return err
	}
	req, err := http.NewRequest("POST", master + path, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Mesos-Stream-Id", mesosStreamID)
	req.Header.Set("Content-Type", "application/json")
	log.Printf("Call %s %s", message.Type, string(body))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("%v Error: %v", log.Llongfile, err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 202 {
		io.Copy(os.Stderr, res.Body)
		return fmt.Errorf("Error %d", res.StatusCode)
	}

	return nil
}
