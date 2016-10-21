This is simeple framewrok aiming to show how to develop Mesos frameworks
against HTTP API.

1. Download the code `git clone https://github.com/janisz/mesos-cookbook.git`
2. Change dir `cd mesos-cookbook/4_understanding_frameworks`
3. In `scheduler.go` change `master` for your URL
4. If you want to mimic some other framework create `/tmp/framework.json` and fill
it with `FrameworkInfo` data:
        
        {
              "id": "<mesos-framewokr-id>",
              "user": "<framework-user>",
              "name": "<framework-name>",
              "failover_timeout": 3600,
              "checkpoint": true,
              "hostname": "<hostname>",
              "webui_url": "<framework-web-ui>"
        }

3. Run it `go run scheduler.go  scheduler.pb.go mesos.pb.go`
