package dispatchsim

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

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
	Distance        float64
}

// there is one and only one order retrieve in the simulation.
type OrderManager struct {
	S             *Simulation
	OrderQueue    chan Order
	Recieve       chan RecieveFormat
	UpdatedOrders []Order
}

func SetupOrderRetrieve(s *Simulation) OrderManager {
	return OrderManager{
		S:             s,
		OrderQueue:    make(chan Order, 5000),
		Recieve:       make(chan RecieveFormat),
		UpdatedOrders: make([]Order, 0),
	}
}

// func (om *OrderManager) runOrderRetrieve() {
// 	csvFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/didi_empty.csv")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	reader := csv.NewReader(csvFile)
// 	var orders []Order
// 	for {
// 		line, error := reader.Read()
// 		if error == io.EOF {
// 			break
// 		} else if error != nil {
// 			fmt.Printf("Error in reading file. \n")
// 			log.Fatal(error)
// 		}
// 		orders = append(orders, Order{
// 			Id:            line[0],
// 			RideStartTime: line[1],
// 			RideStopTime:  line[2],
// 			PickUpLng:     ParseFloatResult(line[3]),
// 			PickUpLat:     ParseFloatResult(line[4]),
// 			DropOffLng:    ParseFloatResult(line[5]),
// 			DropOffLat:    ParseFloatResult(line[6]),
// 		})
// 	}

// 	// sort the orders' timestamp in increasing order (oldest will be at the top)
// 	sort.SliceStable(orders, func(i, j int) bool {
// 		return orders[i].RideStartTime < orders[j].RideStartTime
// 	})

// 	var correctedTasks int = 0
// 	om.S.SendMessageToClient("Loading orders...")
// 	for i := 0; i < len(orders); i++ {
// 		fmt.Printf("[OrderRetriever]Order %v, Time: %v\n", orders[i].Id, orders[i].RideStartTime)
// 		o := SendFormat{3, 1, OrderInfoFormat{1, []float64{orders[i].PickUpLng, orders[i].PickUpLat}, []float64{orders[i].DropOffLng, orders[i].DropOffLat}}}
// 		om.S.Send <- structToString(o)
// 		select {
// 		case r := <-om.Recieve:
// 			//fmt.Printf("Recieve new location %v\n", r.Data.(CorrectedLocation))
// 			sc := r.Data.(CorrectedLocation).StartCoordinate
// 			ec := r.Data.(CorrectedLocation).EndCoordinate
// 			distance := r.Data.(CorrectedLocation).Distance

// 			if (sc == LatLng{} && ec == LatLng{}) {
// 				fmt.Printf("[OrderRetriever]No waypoint available for this order %v\n", orders[i].Id)
// 			} else {
// 				orders[i].StartCoordinate = sc
// 				orders[i].EndCoordinate = ec
// 				orders[i].Distance = distance
// 				om.UpdatedOrders = append(om.UpdatedOrders, orders[i])
// 				correctedTasks++
// 			}
// 		}

// 		//om.S.OrderQueue <- orders[i]
// 	}

// 	// orderTime := ConvertUnixToTimeStamp(om.UpdatedOrders[0].RideStartTime)
// 	// fmt.Printf("[OrderRetriever]First time %v", orderTime)

// 	// // for _, v := range om.UpdatedOrders {
// 	// // 	orderTime := ConvertUnixToTimeStamp(v.RideStartTime)
// 	// // 	fmt.Printf("[OrderRetriever]Order %v - %v \n", v.Id, orderTime)
// 	// // }

// 	// setting up the simulation time
// 	fot := ConvertUnixToTimeStamp(om.UpdatedOrders[0].RideStartTime)
// 	om.S.SimulationTime = time.Date(fot.Year(), fot.Month(), fot.Day(), fot.Hour(), fot.Minute(), fot.Second(), 0, time.Local)
// 	om.S.SimulationTime = om.S.SimulationTime.Add(-3 * time.Minute) // minutes 3 minutes

// 	om.S.SendMessageToClient(strconv.Itoa(correctedTasks) + " Orders loaded")
// 	fmt.Printf("[OrderRetriever]Done \n")
// }

func (om *OrderManager) runOrderRetrieve() {
	csvFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/didi1.csv")
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(csvFile)
	var orders []Order
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Printf("Error in reading file. \n")
			log.Fatal(error)
		}
		orders = append(orders, Order{
			Id:            line[0],
			RideStartTime: line[1],
			RideStopTime:  line[2],
			PickUpLng:     ParseFloatResult(line[3]),
			PickUpLat:     ParseFloatResult(line[4]),
			DropOffLng:    ParseFloatResult(line[5]),
			DropOffLat:    ParseFloatResult(line[6]),
		})
	}

	// sort the orders' timestamp in increasing order (oldest will be at the top)
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].RideStartTime < orders[j].RideStartTime
	})

	var correctedTasks int = 0
	om.S.SendMessageToClient("Loading orders...")
	for i := 0; i < len(orders); i++ {
		fmt.Printf("[OrderRetriever]Order %v, Time: %v\n", orders[i].Id, orders[i].RideStartTime)
		refinedStartCoordinate := om.S.RN.G_FindNearestPoint(LatLng{Lat: orders[i].PickUpLat, Lng: orders[i].PickUpLng})
		refinedEndCoordinate := om.S.RN.G_FindNearestPoint(LatLng{Lat: orders[i].DropOffLat, Lng: orders[i].DropOffLng})
		_, _, distance, _ := om.S.RN.G_GetWaypoint(refinedStartCoordinate, refinedEndCoordinate)
		// fmt.Printf("%v %v %v\n", refinedStartCoordinate, refinedEndCoordinate, distance)
		if (refinedStartCoordinate == LatLng{} && refinedEndCoordinate == LatLng{} || distance == math.Inf(1)) {
			fmt.Printf("[OrderRetriever]No waypoint available for this order %v\n", orders[i].Id)
		} else {
			orders[i].StartCoordinate = refinedStartCoordinate
			orders[i].EndCoordinate = refinedEndCoordinate
			orders[i].Distance = distance
			om.UpdatedOrders = append(om.UpdatedOrders, orders[i])
			correctedTasks++
		}
	}

	// setting up the simulation time
	fot := ConvertUnixToTimeStamp(om.UpdatedOrders[0].RideStartTime)
	om.S.SimulationTime = time.Date(fot.Year(), fot.Month(), fot.Day(), fot.Hour(), fot.Minute(), fot.Second(), 0, time.Local)
	om.S.SimulationTime = om.S.SimulationTime.Add(-3 * time.Minute) // minutes 3 minutes

	om.S.SendMessageToClient(strconv.Itoa(correctedTasks) + " Orders loaded")
	fmt.Printf("[OrderRetriever]Done \n")
}

// func (om *OrderManager) runOrderDistributer2() {
// 	time.Sleep(5 * time.Second)
// 	for _, v := range om.UpdatedOrders {
// 		fmt.Printf("[OrderDistributor]Order found - Id:%v\n", v.Id)
// 		go om.sendOrderToEnvironment(v)
// 	}

// }

func (om *OrderManager) runOrderDistributer() {
	fmt.Printf("[OrderDistributor]Start Order Distributer\n")
	for range om.S.Ticker {
		//fmt.Printf("[OrderDistributor]Time:%v\n", om.S.SimulationTime)
		currentTime := om.S.SimulationTime
		if len(om.UpdatedOrders) > 0 {
			//fmt.Printf("[OrderDistributor]No of orders left in the timeline:%v\n", len(om.UpdatedOrders))
			sameOrderCount := 0
			for _, v := range om.UpdatedOrders {
				orderTime := ConvertUnixToTimeStamp(v.RideStartTime) // we use RideStartTime as the start of order
				if (currentTime.Hour() == orderTime.Hour()) && (currentTime.Minute() == orderTime.Minute()) {
					fmt.Printf("[OrderDistributor]Order found - Id:%v, Time:%v\n", v.Id, orderTime)
					go om.sendOrderToEnvironment(v)
					sameOrderCount++
				}
			}

			if sameOrderCount > 0 {
				om.UpdatedOrders = om.UpdatedOrders[sameOrderCount:]
			}

			//fmt.Printf("[OrderDistributor] %v \n", om.UpdatedOrders[0].RideStartTime)
		}
	}
}

//Order will get "lost" when there is no environment fit for the Order or no environment in the simulation at all
func (om *OrderManager) sendOrderToEnvironment(o Order) {
	tasksGiven := false
	if len(om.S.Environments) > 0 {
	K:
		for _, v := range om.S.Environments {

			if isPointInside(v.Polygon, orb.Point{o.StartCoordinate.Lng, o.StartCoordinate.Lat}) {
				fmt.Printf("[SendOrderToEnv]Allocating Task %v to Environment %d\n", o.Id, v.Id)
				v.TotalTasks = v.TotalTasks + 1 // increment totaltasks
				v.GiveTask(o)
				tasksGiven = true
				break K
			}
		}
		// no environment is able to take in this order
		if !tasksGiven {
			// order will get lost.
			fmt.Printf("[SendOrderToEnv]No available environment to take order %v\n", o.Id)
			tasksGiven = false
		}
	} else {
		fmt.Printf("[SendOrderToEnv]No environment created to take order %v\n", o.Id)
	}
}

func ParseFloatResult(f string) float64 {
	s, _ := strconv.ParseFloat(f, 64)
	return s
}

func ConvertUnixToTimeStamp(x string) time.Time {
	i, err := strconv.ParseInt(x, 10, 64)
	if err != nil {
		panic(err)
	}
	tm := time.Unix(i, 0)
	//fmt.Println(tm)
	return tm
}
