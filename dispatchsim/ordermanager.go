package dispatchsim

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
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
	S              *Simulation
	OrderQueue     chan Order
	Recieve        chan RecieveFormat
	UpdatedOrders  []Order
	IsDistributing bool
}

func SetupOrderRetrieve(s *Simulation) OrderManager {
	return OrderManager{
		S:              s,
		OrderQueue:     make(chan Order, 5000),
		Recieve:        make(chan RecieveFormat),
		UpdatedOrders:  make([]Order, 0),
		IsDistributing: false,
	}
}

func (om *OrderManager) runOrderRetrieve2() {
	csvFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/new_orders_first_1000.csv")
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(csvFile)
	var orders []Order
	om.S.SendMessageToClient("Loading orders...")

	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Printf("Error in reading file. \n")
			log.Fatal(error)
		}

		//_, _, distance, _ := om.S.RN.G_GetWaypoint(LatLng{Lng: ParseFloatResult(line[7]), Lat: ParseFloatResult(line[8])}, LatLng{Lng: ParseFloatResult(line[9]), Lat: ParseFloatResult(line[10])})

		o := &Order{
			Id:              line[0],
			RideStartTime:   line[1],
			RideStopTime:    line[2],
			PickUpLng:       ParseFloatResult(line[3]),
			PickUpLat:       ParseFloatResult(line[4]),
			DropOffLng:      ParseFloatResult(line[5]),
			DropOffLat:      ParseFloatResult(line[6]),
			StartCoordinate: LatLng{Lng: ParseFloatResult(line[7]), Lat: ParseFloatResult(line[8])},
			EndCoordinate:   LatLng{Lng: ParseFloatResult(line[9]), Lat: ParseFloatResult(line[10])},
			Distance:        ParseFloatResult(line[11]),
		}

		orders = append(orders, *o)

		//fmt.Printf("[OrderRetriever]Order %v. Start at time: %v\n", o.Id, o.RideStartTime)
	}

	om.UpdatedOrders = orders
	fot := ConvertUnixToTimeStamp(om.UpdatedOrders[0].RideStartTime)
	lot := ConvertUnixToTimeStamp(om.UpdatedOrders[len(om.UpdatedOrders)-1].RideStartTime)
	om.S.SimulationTime = time.Date(fot.Year(), fot.Month(), fot.Day(), fot.Hour(), fot.Minute(), fot.Second(), 0, time.Local)
	om.S.SimulationTime = om.S.SimulationTime.Add(-1 * time.Minute) // minutes 3 minutes

	om.S.SendMessageToClient(strconv.Itoa(len(om.UpdatedOrders)) + " Orders loaded")
	fmt.Printf("[OrderRetriever]Done \n")
	fmt.Printf("[OrderRetriever]First order will be at %v\n", fot)
	fmt.Printf("[OrderRetriever]Last order will be at %v\n", lot)
	//go om.runOrderDistributer()
	go om.RunOrderDistributor()
}

// combine sendOrderToEnvironments with runOrderDistributer
func (om *OrderManager) RunOrderDistributor() {
	fmt.Printf("[OrderDistributor]Distributing orders to environment...\n")

	//fmt.Printf("[OrderDistributor]Time:%v %v\n", om.S.SimulationTime, om.IsDistributing)

	for _, o := range om.UpdatedOrders {
	K:
		for _, e := range om.S.Environments {
			if isPointInside(e.Polygon, orb.Point{o.StartCoordinate.Lng, o.StartCoordinate.Lat}) {
				task := CreateTaskFromOrder(o, e)
				e.TasksTimeline = append(e.TasksTimeline, task)
				e.Tasks[task.Id] = &task
				break K
			}
		}
	}

	for _, e := range om.S.Environments {
		fmt.Printf("[OrderDistributor]Environment %v has %v\n", e.Id, len(e.TasksTimeline))
	}

	fmt.Printf("[OrderDistributor]Finish distributing orders to environment...\n")
}

func (om *OrderManager) runOrderDistributer() {
	fmt.Printf("[OrderDistributor]Start Order Distributer\n")
	for range om.S.Ticker {
		//fmt.Printf("[OrderDistributor]Time:%v %v\n", om.S.SimulationTime, om.IsDistributing)
		om.S.SendMessageToClient(strconv.Itoa(len(om.UpdatedOrders)) + " Orders left in simulation")
		currentTime := om.S.SimulationTime
		if len(om.UpdatedOrders) > 0 {
			//fmt.Printf("[OrderDistributor]No of orders left in the timeline:%v\n", len(om.UpdatedOrders))
			sameOrderCount := 0
		F:
			for _, v := range om.UpdatedOrders {
				orderTime := ConvertUnixToTimeStamp(v.RideStartTime) // we use RideStartTime as the start of order
				if (currentTime.Hour() == orderTime.Hour()) && (currentTime.Minute() == orderTime.Minute()) {
					om.IsDistributing = true
					fmt.Printf("[OrderDistributor]Order %v sent to environment at time %v\n", v.Id, orderTime)
					om.sendOrderToEnvironment(v)
					sameOrderCount++
				} else {

					break F
				}
			}

			if sameOrderCount > 0 {
				om.UpdatedOrders = om.UpdatedOrders[sameOrderCount:]
			}

			//fmt.Printf("[OrderDistributor] No of orders left in simulation: %v\n", len(om.UpdatedOrders))
		} else {
			fmt.Printf("[OrderDistributor] Finish job")
			return
		}
		om.IsDistributing = false
	}
}

//Order will get "lost" when there is no environment fit for the Order or no environment in the simulation at all
func (om *OrderManager) sendOrderToEnvironment(o Order) {
	tasksGiven := false
K:
	for _, v := range om.S.Environments {

		if isPointInside(v.Polygon, orb.Point{o.StartCoordinate.Lng, o.StartCoordinate.Lat}) {
			fmt.Printf("[SendOrderToEnv - OM]Allocating Task %v to Environment %d\n", o.Id, v.Id)
			//v.TotalTasks = v.TotalTasks + 1 // increment totaltasks
			//v.TaskMutex.Lock()
			v.GiveTask(o)
			//v.TaskMutex.Unlock()
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
