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
	"io/ioutil"
)

const master = "http://10.10.10.10:5050"
const path = "/api/v1/scheduler"

var frameworkInfoFile = fmt.Sprintf("%s/%s", os.TempDir(), "framewrok.json")
var stateFile = fmt.Sprintf("%s/%s", os.TempDir(), "state.json")

var marshaller = jsonpb.Marshaler{
	EnumsAsInts: false,
	Indent:      "  ",
	OrigName:    true,
}
var mesosStreamID string
var frameworkInfo FrameworkInfo
var commandChan = make(chan string, 100)
var taskId uint64
var tasksState = make(map[string]*TaskStatus)

func main() {
	user := "root"
	name := "simple_framework"
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	listen := ":9090"
	webuiUrl := fmt.Sprintf("http://%s%s", hostname, listen)
	failoverTimeout := float64(3600)
	checkpoint := true

	log.Printf("Read FrameworkInfo from %s", frameworkInfoFile)
	frameworkJson, err := ioutil.ReadFile(frameworkInfoFile)
	log.Print(string(frameworkJson))
	if err == nil {
		err := jsonpb.UnmarshalString(string(frameworkJson), &frameworkInfo)
		if (err != nil) {
			log.Printf("Error %v. Please delete %s and restart", err, frameworkInfoFile)
		}
	} else {
		log.Printf("Fallback to defaults due to error [%v]", err)
		frameworkInfo = FrameworkInfo{
			User:     &user,
			Name:     &name,
			Hostname: &hostname,
			WebuiUrl: &webuiUrl,
			FailoverTimeout: &failoverTimeout,
			Checkpoint: &checkpoint,
		}
	}

	go func() {
		log.Fatal(subscribe())
	}()
	http.HandleFunc("/", web)
	err = http.ListenAndServe(listen, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func web(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	switch r.Method {
	case "GET":
		stateJson, err := json.Marshal(tasksState) //Can't use proto.Marshal because there is no definition for map
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		w.Header().Add("Content-type", "application/json")
		fmt.Fprint(w, string(stateJson))
	case "POST":
		cmd := r.Form["cmd"][0]
		commandChan <- cmd
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "Scheduled: %s", cmd)
	case "DELETE":
		id := r.Form["id"][0]
		err := kill(id)
		if err != nil {
			fmt.Fprint(w, err)
		} else {
			fmt.Print(w, "KILLED")
		}
	}
}

func subscribe() error {

	subscribeCall := &Call{
		FrameworkId: frameworkInfo.Id,
		Type:      Call_SUBSCRIBE.Enum(),
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
			mesosStreamID = res.Header.Get("Mesos-Stream-Id")
			json, err := marshaller.MarshalToString(&frameworkInfo)
			if (err != nil) {
				return err
			}
			ioutil.WriteFile(frameworkInfoFile, []byte(json), 0644)
			reconcile()
		case Event_HEARTBEAT:
			log.Print("PING")
		case Event_OFFERS:
			log.Printf("Handle offers returns: %v", handleOffers(sub.Offers))
		case Event_UPDATE:
			log.Printf("Handle update returns: %v", handleUpdate(sub.Update))
		}

	}
}

func reconcile() {
	oldState, err := ioutil.ReadFile(stateFile)
	if err == nil {
		err := json.Unmarshal(oldState, &tasksState)
		if err != nil {
			log.Printf("Error on loading previous state %v", err)
		}
	}

	oldTasks := make([]*Call_Reconcile_Task, 0)
	maxId := 0
	for _, t := range tasksState {
		oldTasks = append(oldTasks, &Call_Reconcile_Task{
			TaskId: t.TaskId,
			AgentId: t.AgentId,
		})
		numericId, err := strconv.Atoi(t.TaskId.GetValue())
		if (err == nil && numericId > maxId) {
			maxId = numericId
		}
	}
	atomic.StoreUint64(&taskId, uint64(maxId))
	call(&Call{
		Type: Call_RECONCILE.Enum(),
		Reconcile: &Call_Reconcile{Tasks: oldTasks},
	})
}

func kill(id string) error {
	update, ok := tasksState[id];
	log.Printf("Kill task %s [%#v]", id, update)
	if !ok {
		return fmt.Errorf("Unknown task %s", id)
	}
	return call(&Call{
		Type: Call_KILL.Enum(),
		Kill: &Call_Kill{
			TaskId: update.TaskId,
			AgentId: update.AgentId,
		},
	})
}

func handleUpdate(update *Event_Update) error {
	tasksState[update.Status.TaskId.GetValue()] = update.Status
	stateJson, _ := json.Marshal(tasksState)
	ioutil.WriteFile(stateFile, stateJson, 0644)
	return call(&Call{
		Type: Call_ACKNOWLEDGE.Enum(),
		Acknowledge: &Call_Acknowledge{
			AgentId: update.Status.AgentId,
			TaskId: update.Status.TaskId,
			Uuid: update.Status.Uuid,
		},
	})
}

func handleOffers(offers *Event_Offers) error {

	offerIds := []*OfferID{}
	for _, offer := range offers.Offers {
		offerIds = append(offerIds, offer.Id)
	}

	select {
	case cmd := <-commandChan:
		firstOffer := offers.Offers[0]

		TRUE := true
		newTaskId := fmt.Sprint(atomic.AddUint64(&taskId, 1))
		accept := &Call{
			Type: Call_ACCEPT.Enum(),
			Accept: &Call_Accept{
				OfferIds: offerIds,
				Operations: []*Offer_Operation{{
					Type: Offer_Operation_LAUNCH.Enum(),
					Launch: &Offer_Operation_Launch{TaskInfos: []*TaskInfo{{
						Name: &cmd,
						TaskId: &TaskID{Value: &newTaskId},
						AgentId: firstOffer.AgentId,
						Resources: defaultResources(),
						Command: &CommandInfo{
							Shell: &TRUE,
							Value: &cmd,
						},
					}, },
					},
				},
				},
			},
		}
		err := call(accept)
		if err != nil {
			return err
		}

	default:
		decline := &Call{
			Type:    Call_DECLINE.Enum(),
			Decline: &Call_Decline{OfferIds: offerIds},
		}
		err := call(decline)
		if err != nil {
			return err
		}
	}
	return nil
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
