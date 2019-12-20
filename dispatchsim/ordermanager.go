package dispatchsim

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
)

type Order struct {
	OrderID       string
	RideStartTime string
	RideStopTime  string
	PickUpLong    string
	PickUpLat     string
	DropOffLong   string
	DropOffLat    string
}

/**
Distribute Task orders from csv, to environment
**/
func OrderDistributor(s *Simulation) {
	var orderId int = 0
	for {
		select {
		case order := <-s.OrderQueue:
			if len(s.Environments) > 0 {
			K:
				for k, v := range s.Environments {
					fmt.Printf("[OrderDistributor]Allocating Task %d to Environment %d\n", orderId, k)
					v.TotalTasks = v.TotalTasks + 1
					v.GenerateTask(orderId)
					orderId++
					break K
				}
			} else {
				//fmt.Println("[OrderDispatcher] Adding back")
				s.OrderQueue <- order
			}
		default:
			//fmt.Printf("Simulation: %d \n", len(s.Environments))
		}

	}
	//fmt.Println("[OrderDispatcher] Stop")
}

/**
Retrieve order from csv.
**/
func OrderRetriever(s *Simulation) {
	csvFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/order1.csv")
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(csvFile)
	var order []Order
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Printf("asdasd\n")
			log.Fatal(error)
		}
		order = append(order, Order{
			OrderID:       line[0],
			RideStartTime: line[1],
			RideStopTime:  line[2],
			PickUpLong:    line[3],
			PickUpLat:     line[4],
			DropOffLong:   line[5],
			DropOffLat:    line[6],
		})
	}

	for i := 0; i < len(order); i++ {
		// i, _ := strconv.ParseInt(order[i].RideStartTime, 10, 64)
		// tm := time.Unix(i, 0)

		fmt.Printf("[OrderRetriever]Order %v\n", order[i].OrderID)
		s.OrderQueue <- order[i].OrderID
	}
}
