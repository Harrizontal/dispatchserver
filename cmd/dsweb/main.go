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

	// not working...
	sendMessage := "Client Connected"
	if err := ws.WriteMessage(1, []byte(sendMessage)); err != nil {
		log.Println(err)
		return
	}
}

func reader(conn *websocket.Conn) {
	for {
		// read in a message
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Error\n")
			log.Println(err)
			return
		}
		//fmt.Println("[Server-Recieve]" + string(p))
		simulator.Recieve <- string(p)
	}
}

func writer(conn *websocket.Conn) {
	for {
		select {
		case sendMessage := <-simulator.Send:
			if err := conn.WriteMessage(1, []byte(sendMessage)); err != nil {
				log.Println(err)
				return
			}
			//default:
		}
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

var simulator = dispatchsim.SetupSimulation()

// Main function to run the simulation
// go run github.com/harrizontal/dispatchserver/cmd/dsweb
func main() {
	fmt.Println("[Server] Proceed to run simulation...")
	go simulator.Run()
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
