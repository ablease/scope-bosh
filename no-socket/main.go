package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type cf_private struct {
	Netmask string
	Ip      string
	Gateway string
}

type networks struct {
	Cf_private cf_private
}

type spec struct {
	Deployment string
	Index      int
	Networks   networks
}

func main() {
	hostID, _ := os.Hostname()

	plugin := &Plugin{HostID: hostID}
	http.HandleFunc("/report", plugin.Report)
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Printf("[Scope-Bosh-Plugin] Http.Serve Error: %v", err)
	}
}

type Plugin struct {
	HostID string
}

func (p *Plugin) Report(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.String())
	specFile, err := ioutil.ReadFile("./example-spec.json")
	if err != nil {
		fmt.Printf("Error reading spec.json: %v\n", err)
		os.Exit(1)
	}
	log.Printf("[Scope-Bosh-Plugin THIS IS TEH SPECFILE: %v\n", string(specFile))

	log.Printf("[Scope-Bosh-Plugin] Report Success: %v", specFile)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(specFile)
}
