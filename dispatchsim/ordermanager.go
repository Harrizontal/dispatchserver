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

// Structure of the Order
// Order is then converted into Task
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

// There is one and ONLY one Order Manager (or Order Retriever) in the Simulation.
type OrderManager struct {
	S              *Simulation
	OrderQueue     chan Order
	Recieve        chan RecieveFormat
	UpdatedOrders  []Order
	IsDistributing bool
}

// Intitalize the Order Retriever
func SetupOrderRetrieve(s *Simulation) OrderManager {
	return OrderManager{
		S:              s,
		OrderQueue:     make(chan Order, 5000),
		Recieve:        make(chan RecieveFormat),
		UpdatedOrders:  make([]Order, 0),
		IsDistributing: false,
	}
}

// This function run the Order Retriver process
// It will read the orders (ride data) and convert into an Order object
// Please note that the .csv file must be preprocessed first, before using implementing the file
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
			fmt.Printf("[Order Retriever]Error in reading file. \n")
			log.Fatal(error)
		}

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

// This function will distributed the order to the environment (or boundaries)
// This function is not used - See RunOrderDistrubotrVirus
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

// This function will distributed the order to the environment (or boundaries)
// This function is USED as it is split the proportion of the virus infected or not infected in tasks, and the masks wore by passengers
func (om *OrderManager) RunOrderDistributorVirus() {
	fmt.Printf("[OrderDistributor]Distributing orders to environment...\n")

	infectedTasks := math.Round(float64(om.S.VirusParameters.InfectedTaskPercentage / 10))
	infectedTasksWithMasks := math.Round(float64(float64(om.S.VirusParameters.PassengerMask)/100.00) * infectedTasks)
	tasksWithMasks := math.Round(float64(float64(om.S.VirusParameters.PassengerMask)/100.00) * float64(10-int(infectedTasks)))
	fmt.Printf("[OrderDistributor]Every 10, there is %v infected tasks\n", infectedTasks)
	fmt.Printf("[OrderDistributor]Every %v of infected tasks, there is %v tasks wearing mask %v\n", infectedTasks, infectedTasksWithMasks, float64(float64(om.S.VirusParameters.PassengerMask)/100))
	fmt.Printf("[OrderDistributor]Every %v healthy tasks, there is %v tasks wearing masks\n", float64(10-int(infectedTasks)), tasksWithMasks)
	i := 0

	// For each order, we distribute the order to TasksTimeline in the environment
	// TasksTimeLine holds the list of order in sequential to the time.
	for _, o := range om.UpdatedOrders {
	K:
		for _, e := range om.S.Environments {
			// If the order is in the boundary, let's distribute the order to the boundary.

			if isPointInside(e.Polygon, orb.Point{o.StartCoordinate.Lng, o.StartCoordinate.Lat}) {
				if math.Mod(float64(i), 10) <= infectedTasks-1 { // task with virus
					if math.Mod(float64(i), infectedTasks) <= float64(infectedTasksWithMasks-1) { // task with mask
						task := CreateVirusTaskFromOrder(o, e, Mild, true) // Create task with Mild virus level and with mask
						fmt.Printf("%v Task - Virus %v, Mask %v\n", i, task.Virus, task.Mask)
						e.TasksTimeline = append(e.TasksTimeline, task)
						e.Tasks[task.Id] = &task
					} else {
						task := CreateVirusTaskFromOrder(o, e, Mild, false) // task w/o mask
						fmt.Printf("%v Task - Virus %v, Mask %v\n", i, task.Virus, task.Mask)
						e.TasksTimeline = append(e.TasksTimeline, task)
						e.Tasks[task.Id] = &task
					}
				} else { // tasks with NO virus
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

	// Once the orders (or tasks in this case, because its converted from Order to task via CreateVirusTaskFromOrder) is distributed to the
	// environment, lets update the TotalTaskToBeCompleted in the environment.
	// This will let the simulation know when to stop the simulation
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
