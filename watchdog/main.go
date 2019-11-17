package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// maps from serverId to containerId
var containerMap map[string]string

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// createInstance creates a new docker instance for each request and loads the game
// for each user separately
func createInstance(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	p := make([]byte, 5)
	username := r.FormValue("username")
	game := r.FormValue("game")

	containerName := randString(6)

	pwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	cmd := exec.Command("docker", "run", "--net=host", "--name", containerName, "-e", "GAME=/dosgames/"+game, "-v", pwd+"/dosgames/"+username+":/dosgames", "game-server")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		cmd.Run()
	}()
	stdout.Read(p)
	containerMap[string(p)] = containerName
	w.Write(p)
}

// stopInstance takes in a query parameter named id and stops the docker
// instance of the server with that serverId
func stopInstance(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	serverId := r.URL.Query().Get("id")
	containerId := containerMap[serverId]
	fmt.Println(containerId)
	cmd := exec.Command("docker", "stop", containerId)
	go func() {
		cmd.Run()
	}()
}

func main() {
	containerMap = make(map[string]string)
	http.HandleFunc("/create", createInstance)
	http.HandleFunc("/stop", stopInstance)
	log.Fatal(http.ListenAndServe(":7000", nil))
}
