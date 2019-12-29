package dispatchsim

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

type Order struct {
	Id            string
	RideStartTime string
	RideStopTime  string
	PickUpLng     float64
	PickUpLat     float64
	DropOffLng    float64
	DropOffLat    float64
}

/**
Distribute Task orders from csv, to environment
**/
func OrderDistributor(s *Simulation) {
	for {
		select {
		case order := <-s.OrderQueue:
			if len(s.Environments) > 0 {
			K:
				for k, v := range s.Environments {
					fmt.Printf("[OrderDistributor]Allocating Task %d to Environment %d\n", order.Id, k)
					v.TotalTasks = v.TotalTasks + 1 // increment totaltasks
					v.GiveTask(order)
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
	csvFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/singaporeorder.csv")
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
			Id:            line[0],
			RideStartTime: line[1],
			RideStopTime:  line[2],
			PickUpLng:     ParseFloatResult(line[3]),
			PickUpLat:     ParseFloatResult(line[4]),
			DropOffLng:    ParseFloatResult(line[5]),
			DropOffLat:    ParseFloatResult(line[6]),
		})
	}

	for i := 0; i < len(order); i++ {
		// i, _ := strconv.ParseInt(order[i].RideStartTime, 10, 64)
		// tm := time.Unix(i, 0)

		fmt.Printf("[OrderRetriever]Order %v\n", order[i].Id)
		s.OrderQueue <- order[i]
	}
}

func ParseFloatResult(f string) float64 {
	s, _ := strconv.ParseFloat(f, 64)
	return s
}
