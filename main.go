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
	err := os.MkdirAll(filepath.Dir(socketPath), 0700)
	if err != nil {
		return nil, fmt.Errorf("[Scope-Bosh-Plugin] Failed to create directory %q: %v", filepath.Dir(socketPath), err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("[Scope-Bosh-Plugin] Failed to listen on %q: %v", socketPath, err)
	}

	log.Printf("[Scope-Bosh-plugin] Listening on: unix://%s", socketPath)
	return listener, nil
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

type Plugin struct {
	HostID string
}

func (p *Plugin) Report(w http.ResponseWriter, r *http.Request) {
	var Spec spec
	log.Println(r.URL.String())

	// Check we can get the bosh /var/vcap/bosh/spec.json
	//Read /var/vcap/bosh/spec.json
	specFile, err := ioutil.ReadFile("./example-spec.json")
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

	response, err := json.Marshal(&Spec)
	if err != nil {
		log.Printf("[Scope-Bosh-Plugin Error Marshalling response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[Scope-Bosh-Plugin] Report Success: %v", response)
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}
