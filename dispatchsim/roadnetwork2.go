package dispatchsim

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/quadtree"
	"github.com/starwander/goraph"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/simple"
)

// type RoadNetwork struct {
// 	Nodes          map[int64]*Node
// 	Links          map[int64]*Link
// 	RoadGraph      *goraph.Graph
// 	QuadTree       *quadtree.Quadtree
// 	RoadGraphGonum *simple.WeightedUndirectedGraph
// }

// type OSM struct {
// 	Version   string `json:"version"`
// 	Generator string `json:"generator"`
// 	Elements  []interface{}
// }

// type OSMNode struct {
// 	Type string  `json:"type"`
// 	Id   int64   `json:"id"`
// 	Lat  float64 `json:"lat"`
// 	Lon  float64 `json:"lon"`
// 	Tags OSMTags `json:"tags"`
// }

// type OSMTags struct {
// 	Highway string
// }

// type OSMWay struct {
// 	Type  string  `json:"type"`
// 	Id    int64   `json:"id"`
// 	Nodes []int64 `json:"nodes"`
// }

// type Node struct {
// 	Id  int64
// 	Lat float64
// 	Lon float64
// }

// type Link struct {
// 	Id     int64 // Way's id
// 	FromId int64
// 	ToId   int64
// }

func SetupRoadNetwork2() *RoadNetwork {

	rn := &RoadNetwork{
		Nodes:          make(map[int64]*Node),  // Node.Id => Node{id,lat,lon}
		NodesCoord:     make(map[string]int64), //lat,lng => Node.Id
		NodesInt:       make(map[int64]*Node),
		Links:          make(map[int64]*Link),
		RoadGraph:      goraph.NewGraph(),
		RoadGraphGonum: simple.NewWeightedUndirectedGraph(0, math.Inf(1)),
	}

	// process osm (json form) to and populate node and links
	rn.G_ProcessJSONFile()

	// populate graph
	rn.G_PopulateGraph()

	// generate quad tree
	rn.G_GenerateQuadTree()

	// test waypoint
	// startTime := time.Now()
	// start := LatLng{Lat: 30.6793691, Lng: 104.0500634}
	// end := LatLng{Lat: 30.6796886, Lng: 104.0505911}
	// rn.G_GetWaypoint(start, end)

	// get random start, end and waypoints
	// for i := 0; i < 1000; i++ {
	// 	start, end, waypoints := rn.G_GetStartEndWaypoint()
	// 	fmt.Printf("start: %v, end: %v, waypoints: %v\n", start, end, len(waypoints))
	// }
	// elapsed := time.Since(startTime)
	// log.Printf("Waypoint generation took %s", elapsed)
	// x := LatLng{Lat: 30.6796886, Lng: 104.0505911}
	// rn.G_GetNextPoint(x)

	// s := LatLng{Lat: 30.6226523, Lng: 104.0678893}
	// s2 := LatLng{Lat: 30.6226523, Lng: 104.0678893}
	// // e := LatLng{Lat: 30.6795686, Lng: 104.094215}

	// start, end, distance, waypoints := rn.G_GetWaypoint(s, s2)
	// fmt.Printf("%v %v %v %v", start, end, distance, waypoints)
	return rn
}

func (rn *RoadNetwork) G_ProcessJSONFile() {
	jsonFile, err := os.Open("./src/github.com/harrizontal/dispatchserver/assets/chengdu_3_osm.json")
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
	var countNode int64 = 0
	for _, element := range osm.Elements {
		e := element.(map[string]interface{})
		jsonbody, _ := json.Marshal(e)
		switch e["type"] {
		case "node": // process node
			oNode := OSMNode{}
			json.Unmarshal(jsonbody, &oNode)
			node := &Node{Id: oNode.Id, Lat: oNode.Lat, Lon: oNode.Lon}
			rn.Nodes[oNode.Id] = node     // Node.Id => Node{id,lat,lon}
			rn.NodesInt[countNode] = node // countnode => node{id,lat,lon}
			c := strconv.FormatFloat(oNode.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(oNode.Lon, 'f', -1, 64)
			rn.NodesCoord[c] = oNode.Id // lat,lng => Node.Id
			countNode++
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

func testGonum() {
	g := simple.NewWeightedUndirectedGraph(0, math.Inf(1))
	var x1, x2, x3, x4, x5 simple.Node = 1, 2, 3, 4, 5
	// g.AddNode(x)
	// g.AddNode(x2)
	// g.AddNode(x3)
	// g.AddNode(x4)
	y := simple.WeightedEdge{F: x1, T: x2, W: 12.0}
	y2 := simple.WeightedEdge{F: x2, T: x3, W: 12.0}
	y3 := simple.WeightedEdge{F: x1, T: x4, W: 12.0}
	y4 := simple.WeightedEdge{F: x3, T: x4, W: 10}
	y5 := simple.WeightedEdge{F: x4, T: x5, W: 1}
	g.SetWeightedEdge(y)
	g.SetWeightedEdge(y2)
	g.SetWeightedEdge(y3)
	g.SetWeightedEdge(y4)
	g.SetWeightedEdge(y5)

	// a, e := path.AStar(x1, x5, g, rn.returnR)
	// fmt.Println(a.To(4))
	// fmt.Println(e)
	// fmt.Println(a.From())
	// fmt.Println(g.Node(5).ID)

}

func (rn *RoadNetwork) returnR(x, y graph.Node) float64 {
	fromNode := rn.Nodes[x.ID()] // get node from map
	toNode := rn.Nodes[y.ID()]   // get node from map
	fromLatLng := LatLng{Lat: fromNode.Lat, Lng: fromNode.Lon}
	toLatLng := LatLng{Lat: toNode.Lat, Lng: toNode.Lon}
	distance := G_calculateDistance(fromLatLng, toLatLng)
	return distance / 2
}

func (rn *RoadNetwork) G_PopulateGraph() {

	// for _, n := range rn.Nodes {
	// 	fmt.Printf("%v\n", n.Id)
	// }

	for _, l := range rn.Links {
		fromNode := rn.Nodes[l.FromId] // get node from map
		toNode := rn.Nodes[l.ToId]     // get node from map
		fromLatLng := LatLng{Lat: fromNode.Lat, Lng: fromNode.Lon}
		toLatLng := LatLng{Lat: toNode.Lat, Lng: toNode.Lon}
		distance := G_calculateDistance(fromLatLng, toLatLng)
		e := simple.WeightedEdge{F: simple.Node(l.FromId), T: simple.Node(l.ToId), W: distance}
		rn.RoadGraphGonum.SetWeightedEdge(e)
	}

	fmt.Printf("[RoadNetwork]Graph populated - nodes:%v links:%v\n", len(rn.Nodes), len(rn.Links))

	// start2 := time.Now()
	// test graph
	// for i := 0; i < 2000; i++ {
	// 	var to int64 = rn.GetRandomNode().Id
	// 	var from int64 = rn.GetRandomNode().Id
	// 	_, e := path.AStar(simple.Node(to), simple.Node(from), rn.RoadGraphGonum, rn.returnR)
	// 	//fmt.Println(a.To(from))
	// 	fmt.Printf("%v, %v\n", i, e)
	// }
	// elapsed2 := time.Since(start2)
	//log.Printf("Waypoint generation took %s", elapsed2)

	// for i := 0; i < 2000; i++ {
	// 	fmt.Println(rn.RoadGraphGonum.From(rn.GetRandomNode().Id))
	// }

}

func (rn *RoadNetwork) G_GenerateQuadTree() {
	arrayPoints := make([]orb.Point, 0)
	for _, node := range rn.Nodes {
		arrayPoints = append(arrayPoints, orb.Point{node.Lon, node.Lat})
	}
	max := G_getMaxBound(arrayPoints)
	min := G_getMinBound(arrayPoints)

	rn.QuadTree = quadtree.New(orb.Bound{Min: min, Max: max})
	for _, point := range arrayPoints {
		//fmt.Println(point)
		rn.QuadTree.Add(point)
	}
	// nearest := rn.QuadTree.Find(orb.Point{104.0554, 30.6522726})

	// fmt.Printf("nearest: %+v\n", nearest)
	fmt.Printf("[RoadNetwork]Quadtree generated\n")
}

func (rn *RoadNetwork) G_FindNearestPoint(ll LatLng) LatLng {
	// take note: orb.Point{lon,lat}
	nearest := rn.QuadTree.Find(orb.Point{ll.Lng, ll.Lat})
	// fmt.Printf("nearest: %+v\n", nearest)
	return LatLng{Lat: nearest.Point().Lat(), Lng: nearest.Point().Lon()}
}

func (rn *RoadNetwork) G_GetRandomNode() *Node {
	//r := rand.Intn(len(rn.Nodes))
	r := GenerateRandomValue(0, len(rn.Nodes))
	// for _, node := range rn.Nodes {
	// 	if r == 0 {
	// 		return node
	// 	}
	// 	r--
	// }
	return rn.NodesInt[int64(r)]
	panic("unreachable")
}

func (rn *RoadNetwork) G_GetRandomLocation() (start LatLng) {
	node := rn.G_GetRandomNode()
	start = LatLng{Lat: node.Lat, Lng: node.Lon}
	return
}

// get a random start, random destination and its waypoint
// [2,1,eId,dId,results[0],results[1],results[2]]
func (rn *RoadNetwork) G_GetStartEndWaypoint() (start, end LatLng, waypoints []LatLng) {
	startNode := rn.G_GetRandomNode()
	endNode := rn.G_GetRandomNode()

	start = LatLng{Lat: startNode.Lat, Lng: startNode.Lon}
	end = LatLng{Lat: endNode.Lat, Lng: endNode.Lon}
	_, _, distance, wy := rn.G_GetWaypoint(start, end)

	waypoints = wy
	//fmt.Printf("[GSEW]No of Waypoint: %v\n", len(wy))
	if distance != math.Inf(1) || len(wy) != 0 {
		return start, end, wy
	}
	//fmt.Printf("[GSEW]Distance is inf or no waypoint available. Calling function\n")
	return rn.G_GetStartEndWaypoint()
}

// get a random destination, and its waypoint from start to the randomed destination
func (rn *RoadNetwork) G_GetEndWaypoint(starting LatLng) (start, end LatLng, waypoints []LatLng) {
	endNode := rn.GetRandomNode()

	start = starting
	end = LatLng{Lat: endNode.Lat, Lng: endNode.Lon}

	// if both same point
	if start == end {
		rn.G_GetEndWaypoint(start)
	}

	_, _, distance, wy := rn.G_GetWaypoint(start, end)

	if distance == math.Inf(1) || len(wy) == 0 {
		return rn.G_GetStartEndWaypoint()
	}

	waypoints = wy
	return
}

// get waypoint between two location
func (rn *RoadNetwork) G_GetWaypoint(starting, ending LatLng) (start, end LatLng, distance float64, waypoints []LatLng) {
	startPoint := strconv.FormatFloat(starting.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(starting.Lng, 'f', -1, 64)
	endPoint := strconv.FormatFloat(ending.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(ending.Lng, 'f', -1, 64)

	// TODO: check whether both point same

	startNode := simple.Node(rn.NodesCoord[startPoint])
	endNode := simple.Node(rn.NodesCoord[endPoint])
	a, _ := path.AStar(startNode, endNode, rn.RoadGraphGonum, rn.returnR)
	nodes, distance := a.To(endNode.ID())
	//fmt.Printf("[GWP]distance: %v , nodes: %v\n", distance, len(nodes))
	waypoints = make([]LatLng, 0)
	for _, node := range nodes {
		//fmt.Println(node)
		//fmt.Printf("Lat:%v, Lng:%v\n", rn.Nodes[node.ID()].Lat, rn.Nodes[node.ID()].Lon)
		waypoints = append(waypoints, LatLng{Lat: rn.Nodes[node.ID()].Lat, Lng: rn.Nodes[node.ID()].Lon})
	}

	//fmt.Printf("[GWP]No of Waypoint: %v\n", len(waypoints))
	//fmt.Printf("Dikjstra: The distance from %v to %v is %v\n", startPoint, endPoint, dist[endPoint])
	return starting, ending, distance, waypoints
}

// go a random next node
func (rn *RoadNetwork) G_GetNextPoint(starting LatLng) (nextLocation LatLng) {
	startPoint := strconv.FormatFloat(starting.Lat, 'f', -1, 64) + "," + strconv.FormatFloat(starting.Lng, 'f', -1, 64)
	startNode := simple.Node(rn.NodesCoord[startPoint])
	x := rn.RoadGraphGonum.From(startNode.ID())
	//fmt.Printf("x: %v\n", x)
	if x.Len() == 0 {
		return LatLng{}
	}

	var count int = 0
	selected := GenerateRandomValue(0, x.Len()-1)
	for {
		item := x.Next()
		if item == false {
			break
		}

		if count == selected {
			selectedNode := rn.Nodes[x.Node().ID()]
			//fmt.Printf("%v\n", selectedNode.Id)
			return LatLng{Lat: selectedNode.Lat, Lng: selectedNode.Lon}
		}
		count++

	}
	return LatLng{}
}

func G_getMaxBound(arrayPoints []orb.Point) orb.Point {
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

func G_getMinBound(arrayPoints []orb.Point) orb.Point {
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

func G_deg2rad(d float64) float64 {
	return (d * math.Pi / 180)
}

// return distance (in km) between two points
func G_calculateDistance(ll1, ll2 LatLng) float64 {
	// return geo.DistanceHaversine(orb.Point{ll1.Lng, ll1.Lat}, orb.Point{ll2.Lng, ll2.Lat})
	dlong := G_deg2rad(ll1.Lng - ll2.Lng)
	dlat := G_deg2rad(ll1.Lat - ll2.Lat)

	lat1 := G_deg2rad(ll1.Lat)
	lat2 := G_deg2rad(ll2.Lat)

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Sin(dlong/2)*math.Sin(dlong/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := c * 6371
	return d
}

// https://stackoverflow.com/questions/8123049/calculate-bearing-between-two-locations-lat-long
// give bearing in degrees from North
func G_calculateBearing(ll1, ll2 LatLng) float64 {
	dLon := (ll2.Lng - ll1.Lng)
	y := math.Sin(dLon) * math.Cos(ll2.Lat)
	x := math.Cos(ll1.Lat)*math.Sin(ll2.Lat) - math.Sin(ll1.Lat)*math.Cos(ll2.Lat)*math.Cos(dLon)
	brng := math.Atan2(y, x) * 180 / math.Pi
	finalBrng := (360 - (math.Mod((brng + 360), 360)))
	return finalBrng
}
