package dispatchsim

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/quadtree"
	"github.com/starwander/goraph"
	"gonum.org/v1/gonum/graph/simple"
)

type RoadNetwork struct {
	Nodes          map[int64]*Node
	NodesCoord     map[string]int64
	NodesInt       map[int64]*Node
	Links          map[int64]*Link
	RoadGraph      *goraph.Graph
	QuadTree       *quadtree.Quadtree
	RoadGraphGonum *simple.WeightedUndirectedGraph
}

type OSM struct {
	Version   string `json:"version"`
	Generator string `json:"generator"`
	Elements  []interface{}
}

type OSMNode struct {
	Type string  `json:"type"`
	Id   int64   `json:"id"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Tags OSMTags `json:"tags"`
}

type OSMTags struct {
	Highway string
}

type OSMWay struct {
	Type  string  `json:"type"`
	Id    int64   `json:"id"`
	Nodes []int64 `json:"nodes"`
}

type Node struct {
	Id  int64
	Lat float64
	Lon float64
}

type Link struct {
	Id     int64 // Way's id
	FromId int64
	ToId   int64
}

// func testSauer() {
// 	g := graph.New()

// 	// add nodes
// 	for i := range [10]int{} {
// 		g.Add(strconv.Itoa(i))
// 	}

// 	// connect nodes
// 	g.Connect("0", "1", 1)
// 	g.Connect("1", "2", 1)
// 	g.Connect("1", "3", 2) // these two lines make it cheaper to go 1→3
// 	g.Connect("2", "3", 2) // than 1→2→3
// 	g.Connect("3", "4", 1)
// 	g.Connect("4", "5", 1)
// 	g.Connect("5", "6", 1)
// 	g.Connect("6", "7", 1)
// 	g.Connect("6", "8", 2) // these two lines make it cheaper to go 6→8
// 	g.Connect("7", "8", 2) // than 6→7→8
// 	g.Connect("8", "9", 1)

// 	// the heuristic function used here returns the absolute difference between the two ints as a simple guessing technique

// 	path, err := g.ShortestPathWithHeuristic("0", "9", returnR)
// 	if err != nil {
// 		fmt.Println("something went wrong:", err)
// 	}

// 	for _, key := range path {
// 		fmt.Print(key, " ")
// 	}
// 	fmt.Println()
// }

// func returnR(startKey, endKey string) int {
// 	return 1
// }

func SetupRoadNetwork() *RoadNetwork {

	rn := &RoadNetwork{
		Nodes:          make(map[int64]*Node),
		NodesCoord:     make(map[string]int64),
		Links:          make(map[int64]*Link),
		RoadGraph:      goraph.NewGraph(),
		RoadGraphGonum: simple.NewWeightedUndirectedGraph(0, math.Inf(1)),
	}

	// process osm (json form) to and populate node and links
	rn.ProcessJSONFile()

	// populate graph
	rn.PopulateGraph()

	// generate quad tree
	rn.GenerateQuadTree()

	//testGonum()

	//rn.testGraph()
	// x := LatLng{Lat: 30.6591745, Lng: 104.0615799}
	// x2 := LatLng{Lat: 30.6682598, Lng: 104.0688429}
	// rn.GetWaypoint(x, x2)

	// for i := 0; i < 1000; i++ {
	// 	fmt.Println(rn.GetStartEndWaypoint())
	// }

	return rn
}

// return no of unique nodes, and array of links
func (rn *RoadNetwork) ProcessJSONFile() {
	jsonFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/chengdu_2_osm.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatal(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var osm OSM
	json.Unmarshal(byteValue, &osm)
	var countLink int64 = 0
	for _, element := range osm.Elements {
		e := element.(map[string]interface{})
		jsonbody, _ := json.Marshal(e)
		switch e["type"] {
		case "node": // process node
			oNode := OSMNode{}
			json.Unmarshal(jsonbody, &oNode)
			rn.Nodes[oNode.Id] = &Node{Id: oNode.Id, Lat: oNode.Lat, Lon: oNode.Lon}

		case "way": // process way
			oWay := OSMWay{}
			json.Unmarshal(jsonbody, &oWay)
			if len(oWay.Nodes) == 0 {
				break
			}
			for i := 1; i < len(oWay.Nodes); i++ {
				from := oWay.Nodes[i]
				to := oWay.Nodes[i-1]
				rn.Links[countLink] = &Link{Id: oWay.Id, FromId: from, ToId: to} // Links[int] = Link{123123,123125}
				countLink++
			}
		}
	}
}

func (rn *RoadNetwork) PopulateGraph() {
	g := rn.RoadGraph
	for _, n := range rn.Nodes {
		// k := strconv.FormatInt(n.Id, 10) // convert int64 into string
		// g.AddVertex(k, LatLng{Lat: n.Lat, Lng: n.Lon})
		x := strconv.FormatFloat(n.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(n.Lon, 'f', -1, 64)
		//fmt.Printf("%v,%v -> %v\n", n.Lat, n.Lon, x)
		g.AddVertex(x, LatLng{Lat: n.Lat, Lng: n.Lon})
	}

	for _, l := range rn.Links {
		fromNode := rn.Nodes[l.FromId] // get node from map
		toNode := rn.Nodes[l.ToId]     // get node from map
		// fmt.Printf("a %v\n", l.FromId)
		// fmt.Printf("b %v\n", l.ToId)
		// fmt.Printf("c %v\n", fromNode)
		// fmt.Printf("d %v\n", toNode)
		fromLatLng := LatLng{Lat: fromNode.Lat, Lng: fromNode.Lon}
		toLatLng := LatLng{Lat: toNode.Lat, Lng: toNode.Lon}
		distance := calculateDistance(fromLatLng, toLatLng)

		// fromId := strconv.FormatInt(l.FromId, 10)
		// toId := strconv.FormatInt(l.ToId, 10)
		fromId := strconv.FormatFloat(fromNode.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(fromNode.Lon, 'f', -1, 64)
		toId := strconv.FormatFloat(toNode.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(toNode.Lon, 'f', -1, 64)
		//fmt.Printf("%v <-> %v \n", fromId, toId)
		g.AddEdge(fromId, toId, distance, nil)
		g.AddEdge(toId, fromId, distance, nil)
	}

	fmt.Printf("[RoadNetwork]Graph populated\n")

}

func (rn *RoadNetwork) testGraph() {

	g := rn.RoadGraph

	// x := LatLng{Lat: 30.6591712, Lng: 104.0717593}
	// x2 := LatLng{Lat: 30.6591214, Lng: 104.0703818}
	x := "30.6591712,104.0717593"
	x2 := "30.6591214,104.0703818"
	fmt.Println(g.GetVertex(x))
	fmt.Println(g.GetVertex(x2))

	dist, prev, _ := g.Dijkstra(x)
	fmt.Printf("Dikjstra: The distance from %v to %v is %v\n", x, x2, dist[x2])

	node := prev[x2]
	for node != nil {
		fmt.Printf("Previous node: %v\n", node)
		// k, _ := g.GetVertex(node)
		// fmt.Printf("Previous node: %v\n", k)
		node = prev[node]
	}

	// graph := goraph.NewGraph()
	// graph.AddVertex("A", LatLng{1.2, 1.2})
	// graph.AddVertex("B", "hello")
	// graph.AddVertex("C", "hello")
	// graph.AddVertex("D", "hello")
	// graph.AddEdge("A", "B", 1.5, "hello")
	// graph.AddEdge("B", "C", 1.5, nil)
	// graph.AddEdge("C", "D", 0.25, nil)
	// graph.AddEdge("D", "C", 1, nil)
	// graph.AddEdge("C", "B", 1.5, nil)
	// graph.AddEdge("B", "A", 1.5, nil)
	// dist, prev, _ := graph.Dijkstra("D")
	// fmt.Println("Dikjstra: The distance from D to A is ", dist["A"])

	// node := prev["A"]
	// for node != nil {
	// 	fmt.Printf("Previous node: %v\n", node)
	// 	// k, _ := graph.GetVertex(node)
	// 	// fmt.Printf("Previous node: %v\n", k)
	// 	node = prev[node]

	// }
	// fmt.Println(graph.GetVertex("A"))
}

func (rn *RoadNetwork) GenerateQuadTree() {
	arrayPoints := make([]orb.Point, 0)
	for _, node := range rn.Nodes {
		arrayPoints = append(arrayPoints, orb.Point{node.Lon, node.Lat})
	}
	max := getMaxBound(arrayPoints)
	min := getMinBound(arrayPoints)

	rn.QuadTree = quadtree.New(orb.Bound{Min: min, Max: max})
	for _, point := range arrayPoints {
		//fmt.Println(point)
		rn.QuadTree.Add(point)
	}
	// nearest := rn.QuadTree.Find(orb.Point{104.0554, 30.6522726})

	// fmt.Printf("nearest: %+v\n", nearest)
	fmt.Printf("[RoadNetwork]Quadtree generated\n")
}

func (rn *RoadNetwork) FindNearestPoint(ll LatLng) LatLng {
	// take note: orb.Point{lon,lat}
	nearest := rn.QuadTree.Find(orb.Point{ll.Lng, ll.Lat})
	// fmt.Printf("nearest: %+v\n", nearest)
	return LatLng{Lat: nearest.Point().Lat(), Lng: nearest.Point().Lon()}
}

func (rn *RoadNetwork) GetRandomNode() *Node {
	r := rand.Intn(len(rn.Nodes))
	for _, node := range rn.Nodes {
		if r == 0 {
			return node
		}
		r--
	}
	panic("unreachable")
}

// [2,1,eId,dId,results[0],results[1],results[2]]
func (rn *RoadNetwork) GetStartEndWaypoint() (start, end LatLng, waypoints []LatLng) {
	startNode := rn.GetRandomNode()
	endNode := rn.GetRandomNode()

	// if both same point
	if startNode.Id == endNode.Id {
		rn.GetStartEndWaypoint()
	}
	start = LatLng{Lat: startNode.Lat, Lng: startNode.Lon}
	end = LatLng{Lat: endNode.Lat, Lng: endNode.Lon}

	_, _, distance, wy := rn.GetWaypoint(start, end)

	if distance == math.Inf(1) {
		rn.GetStartEndWaypoint()
	}

	waypoints = wy
	return

}

func (rn *RoadNetwork) GetEndWaypoint(starting LatLng) (start, end LatLng, waypoints []LatLng) {
	endNode := rn.GetRandomNode()

	start = starting
	end = LatLng{Lat: endNode.Lat, Lng: endNode.Lon}

	// if both same point
	if start == end {
		rn.GetEndWaypoint(start)
	}

	_, _, distance, wy := rn.GetWaypoint(start, end)

	if distance == math.Inf(1) {
		rn.GetStartEndWaypoint()
	}

	waypoints = wy
	return
}

func (rn *RoadNetwork) GetWaypoint(starting, ending LatLng) (start, end LatLng, distance float64, waypoints []LatLng) {
	g := rn.RoadGraph
	startPoint := strconv.FormatFloat(starting.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(starting.Lng, 'f', -1, 64)
	endPoint := strconv.FormatFloat(ending.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(ending.Lng, 'f', -1, 64)

	start = starting
	end = ending
	// TODO: check whether both point same
	dist, prev, _ := g.Dijkstra(startPoint)

	distance = dist[endPoint]
	waypoints = []LatLng{ending}
	node := prev[endPoint]
	for node != nil {
		//fmt.Printf("Previous node: %v\n", node)
		x := node.(string)
		coord := strings.Split(x, ",") // [0] = lat, [1] = lng
		lat, _ := strconv.ParseFloat(coord[0], 64)
		lng, _ := strconv.ParseFloat(coord[1], 64)
		waypoints = append(waypoints, LatLng{})
		copy(waypoints[1:], waypoints)
		waypoints[0] = LatLng{Lat: lat, Lng: lng}
		node = prev[node]
	}

	//fmt.Println(waypoint)
	//fmt.Printf("Dikjstra: The distance from %v to %v is %v\n", startPoint, endPoint, dist[endPoint])
	return
}

func getMaxBound(arrayPoints []orb.Point) orb.Point {
	var lon float64 = arrayPoints[0].Lon()
	var lat float64 = arrayPoints[0].Lat()
	for _, point := range arrayPoints {
		if lon < point.Lon() {
			lon = point.Lon()
		}
		if lat < point.Lat() {
			lat = point.Lat()
		}
	}
	return orb.Point{lon, lat}
}

func getMinBound(arrayPoints []orb.Point) orb.Point {
	var lon float64 = arrayPoints[0].Lon()
	var lat float64 = arrayPoints[0].Lat()
	for _, point := range arrayPoints {
		if lon > point.Lon() {
			lon = point.Lon()
		}
		if lat > point.Lat() {
			lat = point.Lat()
		}
	}
	return orb.Point{lon, lat}
}

func deg2rad(d float64) float64 {
	return (d * math.Pi / 180)
}

// return distance (in km) between two points
func calculateDistance(ll1, ll2 LatLng) float64 {
	// return geo.DistanceHaversine(orb.Point{ll1.Lng, ll1.Lat}, orb.Point{ll2.Lng, ll2.Lat})
	dlong := deg2rad(ll1.Lng - ll2.Lng)
	dlat := deg2rad(ll1.Lat - ll2.Lat)

	lat1 := deg2rad(ll1.Lat)
	lat2 := deg2rad(ll2.Lat)

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Sin(dlong/2)*math.Sin(dlong/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := c * 6371
	return d
}
