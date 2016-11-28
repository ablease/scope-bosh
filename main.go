package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func setupSignals(socketPath string) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		os.RemoveAll(filepath.Dir(socketPath))
		os.Exit(0)
	}()
}

func setupSocket(socketPath string) (net.Listener, error) {
	os.RemoveAll(filepath.Dir(socketPath))
	fmt.Printf("[Scope-Bosh-Plugin] Attempting to create socket at %v:", socketPath)
	err := os.MkdirAll(filepath.Dir(socketPath), 0700)
	if err != nil {
		return nil, fmt.Errorf("[Scope-Bosh-Plugin] Failed to create directory %q: %v", filepath.Dir(socketPath), err)
	}
	fmt.Printf("[Scope-Bosh-Plugin] Socket Created at %v:", socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("[Scope-Bosh-Plugin] Failed to listen on %q: %v", socketPath, err)
	}

	log.Printf("[Scope-Bosh-plugin] Listening on: unix://%s", socketPath)
	return listener, nil
}

type Plugin struct {
	HostID   string
	boshMode bool
}

type report struct {
	Host    spec
	Plugins []pluginSpec
}

type pluginSpec struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Interfaces  []string `json:"interfaces"`
	APIVersion  string   `json:"api_version,omitempty"`
}

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
	const socketPath = "/var/vcap/packages/scope/plugins/scope-bosh/bosh.sock"
	hostID, _ := os.Hostname()

	setupSignals(socketPath)

	log.Printf("Starting on %s...\n", hostID)

	listener, err := setupSocket(socketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		listener.Close()
		os.RemoveAll(filepath.Dir(socketPath))
	}()

	plugin := &Plugin{HostID: hostID}
	http.HandleFunc("/report", plugin.Report)
	err = http.Serve(listener, nil)
	if err != nil {
		log.Printf("[Scope-Bosh-Plugin] Http.Serve Error: %v", err)
	}
}

func (p *Plugin) Report(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.String())
	rpt, err := p.makeReport()
	if err != nil {
		log.Printf("[Scope-Bosh-Plugin]Error making report: %v\n", err)
		return
	}

	response, err := json.Marshal(*rpt)
	if err != nil {
		log.Printf("[Scope-Bosh-Plugin Error Marshalling response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[Scope-Bosh-Plugin] Report Success: %v", response)
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (p *Plugin) makeReport() (*report, error) {
	var Spec spec
	// Check we can get the bosh /var/vcap/bosh/spec.json
	//Read /var/vcap/bosh/spec.json
	specFile, err := ioutil.ReadFile("/var/vcap/packages/scope/plugins/spec.json")
	if err != nil {
		fmt.Printf("Error reading spec.json: %v\n", err)
		os.Exit(1)
	}
	//Unmarshall spec.json into spec type
	err = json.Unmarshal(specFile, &Spec)
	if err != nil {
		fmt.Printf("Error unmarshalling spec.json: %v\n", err)
	}

	// Validate Umarshalling
	fmt.Printf("Deployment: %+v\n", Spec.Deployment)
	fmt.Printf("Index: %+v\n", Spec.Index)
	fmt.Printf("Networks: %+v", Spec.Networks)

	rpt := &report{
		Host: spec{
			Deployment: Spec.Deployment,
			Index:      Spec.Index,
			Networks:   Spec.Networks,
		},
		Plugins: []pluginSpec{
			{
				ID:          "bosh",
				Label:       "scope-bosh",
				Description: "Displays information about the bosh deployed VM",
				Interfaces:  []string{"reporter"},
				APIVersion:  "1",
			},
		},
	}

	return rpt, nil
}
