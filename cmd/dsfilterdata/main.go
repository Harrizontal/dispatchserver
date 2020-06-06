package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/harrizontal/dispatchserver/dispatchsim"
)

// This file is important for preprocessing the ride data
// The ride data structure is: Id, Ride start time, ride stop time, pick up lng, pick up lat, drop off lng and drop off lat
// Pickup Lat and Lng, and Drop off Lat and Lng are heavily preprocessed towards the simulation.
// If is not preprocessed in the beginning, it will take a lot of CPU usage and memory for the simulation programme to calculate.

// Before you run this file, please input the ride data file (full path) and the name of the csv file (full path)
// I suggest you to split the ride data file into smaller bits first as it will take some time to process.

func main() {
	rn := dispatchsim.SetupRoadNetwork2()
	orders := runOrderRetrieve(rn)
	writeOrdersToFile(orders)
}

func runOrderRetrieve(rn *dispatchsim.RoadNetwork) []dispatchsim.Order {
	csvFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/didi2.csv")
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(csvFile)
	var orders []dispatchsim.Order
	var updatedOrders []dispatchsim.Order
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Printf("Error in reading file. \n")
			log.Fatal(error)
		}
		orders = append(orders, dispatchsim.Order{
			Id:            line[0],
			RideStartTime: line[1],
			RideStopTime:  line[2],
			PickUpLng:     dispatchsim.ParseFloatResult(line[3]),
			PickUpLat:     dispatchsim.ParseFloatResult(line[4]),
			DropOffLng:    dispatchsim.ParseFloatResult(line[5]),
			DropOffLat:    dispatchsim.ParseFloatResult(line[6]),
		})
	}

	// sort the orders' timestamp in increasing order (oldest will be at the top)
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].RideStartTime < orders[j].RideStartTime
	})

	// Generate the refined Start Coordinate, refined End Coordinate and distance between the two coordinate (in terms of road network - not straight line)
	var correctedTasks int = 0
	for i := 0; i < len(orders); i++ {
		fmt.Printf("[OrderRetriever %v]Order %v, Time: %v\n", correctedTasks, orders[i].Id, orders[i].RideStartTime)
		refinedStartCoordinate := rn.FindNearestPoint(dispatchsim.LatLng{Lat: orders[i].PickUpLat, Lng: orders[i].PickUpLng})
		refinedEndCoordinate := rn.FindNearestPoint(dispatchsim.LatLng{Lat: orders[i].DropOffLat, Lng: orders[i].DropOffLng})
		_, _, distance, _ := rn.GetWaypoint(refinedStartCoordinate, refinedEndCoordinate)
		if (refinedStartCoordinate == dispatchsim.LatLng{} && refinedEndCoordinate == dispatchsim.LatLng{} || distance == math.Inf(1)) {
			fmt.Printf("[OrderRetriever %v]No waypoint available for this order %v\n", correctedTasks, orders[i].Id)
		} else {
			orders[i].StartCoordinate = refinedStartCoordinate
			orders[i].EndCoordinate = refinedEndCoordinate
			orders[i].Distance = distance
			updatedOrders = append(updatedOrders, orders[i])
			correctedTasks++
		}
	}

	return updatedOrders
}

// Write the orders to the csv
func writeOrdersToFile(o []dispatchsim.Order) {
	csvFile, err := os.Create("./src/github.com/harrizontal/dispatchserver/assets/new_orders_first_1000.csv")

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvFile)

	count := 1
	total := len(o)
	for _, orderRow := range o {
		fmt.Printf("[Writing]%v/%v\n", count, total)
		data := []string{orderRow.Id,
			orderRow.RideStartTime,
			orderRow.RideStopTime,
			FloatToString(orderRow.PickUpLng),
			FloatToString(orderRow.PickUpLat),
			FloatToString(orderRow.DropOffLng),
			FloatToString(orderRow.DropOffLat),
			FloatToString(orderRow.StartCoordinate.Lng),
			FloatToString(orderRow.StartCoordinate.Lat),
			FloatToString(orderRow.EndCoordinate.Lng),
			FloatToString(orderRow.EndCoordinate.Lat),
			FloatToString(orderRow.Distance)}

		csvwriter.Write(data)
		count++
	}

	csvwriter.Flush()
	csvFile.Close()
}

func FloatToString(input_num float64) string {
	return strconv.FormatFloat(input_num, 'f', -1, 64)
}
