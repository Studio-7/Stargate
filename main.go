package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	// "image/jpeg"
	"strconv"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"

	vpxEncoder "github.com/poi5305/go-yuv2webRTC/vpx-encoder"
	"github.com/poi5305/go-yuv2webRTC/screenshot"
	"github.com/go-vgo/robotgo"
	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	// "github.com/pion/webrtc/pkg/media"
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

const width = 853
const height = 480
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// Calls the signalling server with games and id
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
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	offer := webrtc.SessionDescription{}
	Decode(clientOffer, &offer)

	// mediaEngine := webrtc.MediaEngine{}
	// err := mediaEngine.PopulateFromSDP(offer)
	// if err != nil {
	// 	panic(err)
	// }

	// // Create a new RTCPeerConnection
	// api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Create a video track
	videoTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "pion")
	if err != nil {
		panic(err)
	}
	if _, err = peerConnection.AddTrack(videoTrack); err != nil {
		panic(err)
	}

	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}
	go startEncoding(videoTrack)

	serverSDP := Encode(answer)
	fmt.Println(serverSDP)

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		if d.Label() == "foo" {
			// d.OnOpen(func() {
				// for {
				// 	// start := time.Now()
				// 	data, _ := screenshot.Capture(0, 0, 640, 480)
				// 	go func() {
				// 		buf := new(bytes.Buffer)
				// 		jpeg.Encode(buf, data, nil)
				// 		img := buf.Bytes()
				// 		d.Send(img)
				// 	}()
				// 	// elapsed := time.Since(start)
				// 	// fmt.Println(elapsed)
				// }
				
			// })

			go func() {
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
					coords := strings.Split(string(msg.Data), ",")
					x, _ := strconv.Atoi(coords[0])
					y, _ := strconv.Atoi(coords[1])
					robotgo.Move(x, y)
				})
			}()

		} else if d.Label() == "click" {
			d.OnOpen(func() {
				fmt.Println("Click channel listener attached")
			})

			go func() {
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					action := string(msg.Data)
					switch action {
					case "ld":
						robotgo.MouseClick("left")
					case "rd":
						robotgo.MouseClick("right")
					}
				})
			}()
		} else if d.Label() == "key" {
			d.OnOpen(func() {
				fmt.Println("Key channel listener attached")
			})

			go func() {
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					action := string(msg.Data)
					fmt.Println(action)
					robotgo.KeyTap(action)
				})
			}()
		}

	})

	return serverSDP
}


func startEncoding(videoTrack *webrtc.Track) {
	encoder, err := vpxEncoder.NewVpxEncoder(width, height, 30, 1200, 5)
	if err != nil {
		panic(err)
	}
	// Capture
	go func() {
		for {
			rgbaImg := screenshot.GetScreenshot(0, 0, width, height, width, height)
			yuv := screenshot.RgbaToYuv(rgbaImg)
			encoder.Input <- yuv
		}
	}()
	// Encode
	go func() {
		for {
			bs := <-encoder.Output
			videoTrack.WriteSample(media.Sample{Data: bs, Samples: 900000})
		}
	}()
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
	b, err := base64.StdEncoding.DecodeString(strings.Replace(in, "\n", "", -1))
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
