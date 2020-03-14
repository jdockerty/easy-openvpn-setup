package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
)

type Client struct {
	Name       string
	TLSEncrypt []byte
}

type Status struct {
	Code int
}

// StatusHandler used to check whether the server is running, the 200 OK is returned if it is.
// This provides an easy way to check whether the service is running before constructing a larger POST.
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	newStatusResponse := Status{Code: http.StatusOK}
	json.Marshal(newStatusResponse)
	json.NewEncoder(w).Encode(newStatusResponse)
}

// AddClientHandler is used to add a client to the running OpenVPN server.
// It uses the JSON contained within the POST request method to construct a profile on the server. This responds with the constructed profile to insert
// into the OpenVPN Client UI.
func AddClientHandler(w http.ResponseWriter, r *http.Request) {
	var newClient Client

	requestBody, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(requestBody, &newClient)

	fmt.Println("Recieved new client: ", newClient.Name)
	executeOpenVPNScript(newClient, w)
}

func executeReadNewProfile(clientName string) string {
	// Command for reading the .ovpn config file created on the server.
	TLSCommandString := "sudo cat /root/" + clientName + ".ovpn"

	output, err := exec.Command("bash", "-c", TLSCommandString).Output()
	if err != nil {
		panic(err)
	}
	return string(output)
}

func executeOpenVPNScript(clientToAdd Client, responseWriter http.ResponseWriter) {
	// Command to pipe into the shell script, selects option 1 and adds given client name.
	c1 := exec.Command("printf", fmt.Sprintf("1\n%s", clientToAdd.Name))
	c2 := exec.Command("bash", "-c", "sudo ~/openvpn-install/openvpn-install.sh")

	r, w := io.Pipe()
	c1.Stdout = w // Reader is tied to Stdout of command 1
	c2.Stdin = r  // Writer is tied to Stdin of command 2

	c1.Start() // Start command 1 execution
	c2.Start() // Execute command 2
	c1.Wait()
	w.Close()
	c2.Wait()

	clientResponseData := executeReadNewProfile(clientToAdd.Name)
	fmt.Println(clientResponseData)

	// Write the response to user, allows them to copy/paste the output into an .ovpn file
	responseWriter.Write([]byte(string("Paste the following into an .ovpn file: \n" + clientResponseData)))

}

func main() {
	newRouter := mux.NewRouter()
	fmt.Println("Running...")
	newRouter.HandleFunc("/api/Status", StatusHandler)
	newRouter.HandleFunc("/api/AddClient", AddClientHandler).Methods("POST")
	http.ListenAndServe(":8080", newRouter)
}
