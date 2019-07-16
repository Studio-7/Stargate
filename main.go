package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"

	"time"

	"github.com/joho/godotenv"
	"github.com/kbinani/screenshot"
	"github.com/pion/webrtc"
	"github.com/sacOO7/gowebsocket"
)

type ServerMsg struct {
	Type  int      `json:"type"`
	ID    string   `json:"id"`
	SDP   string   `json:"sdp"`
	Games []string `json:"games"`
}

const compress = false

var serverId string
var socket gowebsocket.Socket
var signalUrl string

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// calls the signalling server with games and id
func signalInit() {
	initMsg := ServerMsg{
		Type:  1,
		ID:    serverId,
		Games: []string{"cs", "cod"},
	}
	initJson, err := json.Marshal(initMsg)
	if err != nil {
		log.Println(err)
	}
	fmt.Println(string(initJson))
	socket.SendText(string(initJson))
}

// Acks the client with the server sdp
func signalAck(serverSDP string) {
	ackMsg := ServerMsg{
		Type: 2,
		ID:   serverId,
		SDP:  serverSDP,
	}
	ackJson, err := json.Marshal(ackMsg)
	if err != nil {
		log.Println(err)
	}
	fmt.Println(string(ackJson))
	socket.SendText(string(ackJson))
}

func serverInit() {
	serverId = randString(5)
	e := godotenv.Load()
	if e != nil {
		log.Fatal(e)
	}
	signalUrl = os.Getenv("SIGNAL")

	socket = gowebsocket.New(signalUrl)

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Fatal("Received connect error - ", err)
	}

	socket.OnConnected = func(socket gowebsocket.Socket) {
		log.Println("Connected to server")
	}

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		// Listen for client request and init webrtc
		// Only one client allowed, first come first serve
		log.Println("Received message - " + message)
		serverSDP := setupWebrtc(message)
		signalAck(serverSDP)
	}

	socket.OnPingReceived = func(data string, socket gowebsocket.Socket) {
		log.Println("Received ping - " + data)
	}

	socket.OnPongReceived = func(data string, socket gowebsocket.Socket) {
		log.Println("Received pong - " + data)
	}

	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		log.Println("Disconnected from server ")
		return
	}

	socket.Connect()
	signalInit()
}

func setupWebrtc(clientOffer string) string {
	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())
			for {
				start := time.Now()
				data, _ := screenshot.Capture(0, 0, 640, 480)
				buf := new(bytes.Buffer)
				jpeg.Encode(buf, data, nil)
				img := buf.Bytes()
				elapsed := time.Since(start)
				fmt.Println(elapsed)
				d.Send(img)
			}
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	Decode(clientOffer, &offer)

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Output the answer in base64 so we can paste it in browser
	serverSDP := Encode(answer)
	fmt.Println(serverSDP)
	return serverSDP
}

func main() {
	serverInit()
	select {}
}

// Encode encodes the input in base64
// It can optionally zip the input before encoding
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		fmt.Println(err)
	}

	if compress {
		b = zip(b)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(in, "\n", ""))
	if err != nil {
		fmt.Println(err)
	}

	if compress {
		b = unzip(b)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		fmt.Println(err)
	}
}

func MustReadStdin() string {
	r := bufio.NewReader(os.Stdin)

	var in string
	for {
		var err error
		in, err = r.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				fmt.Println(err)
			}
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			break
		}
	}

	fmt.Println("")

	return in
}

func zip(in []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(in)
	if err != nil {
		fmt.Println(err)
	}
	err = gz.Flush()
	if err != nil {
		fmt.Println(err)
	}
	err = gz.Close()
	if err != nil {
		fmt.Println(err)
	}
	return b.Bytes()
}

func unzip(in []byte) []byte {
	var b bytes.Buffer
	_, err := b.Write(in)
	if err != nil {
		fmt.Println(err)
	}
	r, err := gzip.NewReader(&b)
	if err != nil {
		fmt.Println(err)
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		fmt.Println(err)
	}
	return res
}
