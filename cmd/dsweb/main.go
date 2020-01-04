package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/harrizontal/dispatchserver/dispatchsim"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	// helpful log statement to show connections
	log.Println("Client Connected")

	if err != nil {
		log.Println(err)
	}

	go reader(ws)
	go writer(ws)
}

func reader(conn *websocket.Conn) {
	for {
		// read in a message
		//fmt.Println("[Server]Waiting for message from client")
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Error\n")
			log.Println(err)
			return
		}
		fmt.Println("[Server-Recieve]" + string(p))
		simulator.Recieve <- string(p)
	}
}

func writer(conn *websocket.Conn) {
	for {
		select {
		case sendMessage := <-simulator.Send:
			//fmt.Println("[Server]Message from simulator sent")
			if err := conn.WriteMessage(1, []byte(sendMessage)); err != nil {
				log.Println(err)
				return
			}
		default:
		}
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

var simulator = dispatchsim.SetupSimulation()

func main() {
	fmt.Println("[Server]Started")
	go simulator.Run()
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
