package dispatchsim

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/paulmach/orb"
)

type Order struct {
	Id              string
	RideStartTime   string
	RideStopTime    string
	PickUpLng       float64
	PickUpLat       float64
	DropOffLng      float64
	DropOffLat      float64
	StartCoordinate LatLng
	EndCoordinate   LatLng
}

// there is one and only one order retrieve in the simulation.
type OrderManager struct {
	S          *Simulation
	OrderQueue chan Order
	Recieve    chan RecieveFormat
}

func SetupOrderRetrieve(s *Simulation) OrderManager {
	return OrderManager{
		S:          s,
		OrderQueue: make(chan Order, 5000),
		Recieve:    make(chan RecieveFormat),
	}
}

func (om *OrderManager) runOrderRetrieve() {
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
		fmt.Printf("[OrderRetriever]Order %v\n", order[i].Id)
		om.S.OrderQueue <- order[i]
	}
}

func (om *OrderManager) runOrderDistribute() {
	tasksGiven := false
	for {
		select {
		case order := <-om.S.OrderQueue:
			// get the start coordinate and end coordinate for routing of the order even though is provided.
			if (order.StartCoordinate == LatLng{} && order.EndCoordinate == LatLng{}) {
				o := SendFormat{3, 1, OrderInfoFormat{1, []float64{order.PickUpLng, order.PickUpLat}, []float64{order.DropOffLng, order.DropOffLat}}}
				fmt.Printf("[Send] %v\n", order)
				om.S.Send <- structToString(o)
				select {
				case r := <-om.Recieve:
					fmt.Printf("Recieve new location %v\n", r.Data.(CorrectedLocation))
					order.StartCoordinate = r.Data.(CorrectedLocation).StartCoordinate
					order.EndCoordinate = r.Data.(CorrectedLocation).EndCoordinate
				}
			}
			// start distribution of the order
			if len(om.S.Environments) > 0 {
			K:
				for _, v := range om.S.Environments {

					if isPointInside(v.Polygon, orb.Point{order.StartCoordinate.Lng, order.StartCoordinate.Lat}) {
						fmt.Printf("[OrderDistributor]Allocating Task %v to Environment %d\n", order.Id, v.Id)
						v.TotalTasks = v.TotalTasks + 1 // increment totaltasks
						v.GiveTask(order)
						tasksGiven = true
						break K
					}
				}
				// no environment is able to take in this order
				if !tasksGiven {
					om.S.OrderQueue <- order // put back to order queue
					tasksGiven = false
				}
			} else {
				// no environment available - put back to order queue
				om.S.OrderQueue <- order
			}
		default:
			//fmt.Printf("Simulation: %d \n", len(s.Environments))
		}

	}
	//fmt.Println("[OrderDispatcher] Stop")
}

func (om *OrderManager) getCoord() {
	// lat lng format... should change to lng lat soon
	o := SendFormat{3, 1, OrderInfoFormat{1, []float64{103.8468822, 1.2783807}, []float64{103.846895, 1.2770392}}}
	om.S.Send <- structToString(o)
	select {
	case <-om.Recieve:
		fmt.Printf("Recieve new location\n")
	}
	fmt.Printf("End new location\n")
}

func ParseFloatResult(f string) float64 {
	s, _ := strconv.ParseFloat(f, 64)
	return s
}
