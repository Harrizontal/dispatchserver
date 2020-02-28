package dispatchsim

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
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

func (om *OrderManager) RunOrderRetriever() {
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
	var newTime time.Time = time.Date(fot.Year(), fot.Month(), fot.Day(), fot.Hour(), fot.Minute(), fot.Second(), 0, time.Local)
	newTime = newTime.Add(-1 * time.Minute)
	om.S.SimulationTime = newTime
	om.S.CaptureSimulationTime = newTime

	om.S.SendMessageToClient(strconv.Itoa(len(om.UpdatedOrders)) + " Orders loaded")
	fmt.Printf("[OrderRetriever]Done \n")
	fmt.Printf("[OrderRetriever]First order will be at %v\n", fot)
	fmt.Printf("[OrderRetriever]Last order will be at %v\n", lot)
	go om.RunOrderDistributorVirus()
}

// combine sendOrderToEnvironments with runOrderDistributer
func (om *OrderManager) RunOrderDistributor() {
	fmt.Printf("[OrderDistributor]Distributing orders to environment...\n")

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

// virus
func (om *OrderManager) RunOrderDistributorVirus() {
	fmt.Printf("[OrderDistributor]Distributing orders to environment...\n")

	infectedTasks := math.Round(float64(om.S.VirusParameters.InfectedTaskPercentage / 10))
	infectedTasksWithMasks := math.Round(float64(float64(om.S.VirusParameters.PassengerMask)/100.00) * infectedTasks)
	tasksWithMasks := math.Round(float64(float64(om.S.VirusParameters.PassengerMask)/100.00) * float64(10-int(infectedTasks)))
	fmt.Printf("[OrderDistributor]Every 10, there is %v infected tasks\n", infectedTasks)
	fmt.Printf("[OrderDistributor]Every %v of infected tasks, there is %v tasks wearing mask %v\n", infectedTasks, infectedTasksWithMasks, float64(float64(om.S.VirusParameters.PassengerMask)/100))
	fmt.Printf("[OrderDistributor]Every %v healthy tasks, there is %v tasks wearing masks\n", float64(10-int(infectedTasks)), tasksWithMasks)
	i := 0
	for _, o := range om.UpdatedOrders {
	K:
		for _, e := range om.S.Environments {
			if isPointInside(e.Polygon, orb.Point{o.StartCoordinate.Lng, o.StartCoordinate.Lat}) {
				if math.Mod(float64(i), 10) <= infectedTasks-1 { // task with virus
					if math.Mod(float64(i), infectedTasks) <= float64(infectedTasksWithMasks-1) { // task with mask
						task := CreateVirusTaskFromOrder(o, e, Mild, true)
						fmt.Printf("%v Task - Virus %v, Mask %v\n", i, task.Virus, task.Mask)
						e.TasksTimeline = append(e.TasksTimeline, task)
						e.Tasks[task.Id] = &task
					} else {
						task := CreateVirusTaskFromOrder(o, e, Mild, false) // task w/o mask
						fmt.Printf("%v Task - Virus %v, Mask %v\n", i, task.Virus, task.Mask)
						e.TasksTimeline = append(e.TasksTimeline, task)
						e.Tasks[task.Id] = &task
					}
				} else { // tasks with virus
					if math.Mod(float64(i), 10) <= infectedTasks+tasksWithMasks-1 { // task with mask
						task := CreateVirusTaskFromOrder(o, e, None, true)
						fmt.Printf("%v Task - Virus %v, Mask %v\n", i, task.Virus, task.Mask)
						e.TasksTimeline = append(e.TasksTimeline, task)
						e.Tasks[task.Id] = &task
					} else {
						task := CreateVirusTaskFromOrder(o, e, None, false) // task w/o mask
						fmt.Printf("%v Task - Virus %v, Mask %v\n", i, task.Virus, task.Mask)
						e.TasksTimeline = append(e.TasksTimeline, task)
						e.Tasks[task.Id] = &task
					}
				}
				i++
				break K
			}
		}
	}

	for _, e := range om.S.Environments {
		fmt.Printf("[OrderDistributor]Environment %v has %v\n", e.Id, len(e.TasksTimeline))
		e.TotalTasksToBeCompleted = len(e.TasksTimeline)

	}

	fmt.Printf("[OrderDistributor]Finish distributing orders to environment...\n")
	om.S.SendMessageToClient("Awaiting to Start")
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
	return tm
}
