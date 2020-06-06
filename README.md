# A Multi-Agent Ride Sharing System Simulation built with Golang (with infectious spread mechanism)
`dispatchserver` is a simulator where it simulate a ride sharing system in Chengdu, China. The simulation can spawn smart driver agents that are able to complete a set of tasks (picking up passengers and fetching passengers to the stated destination). The simulator uses Websocket to communicate with a [map visualization tool](https://github.com/Harrizontal/Intelligent-Order-Matching-Sim)

## Background
The simulator is equipped with a novel order dispatch algorithm in large-scale on-demand ride-hailing platforms
that take account of the dynamic characteristics associated with workers. Although most traditional order dispatch approaches generally focus on providing a better user experience for passengers and maximizing revenue by optimizing resource utilization, the proposed
algorithm is designed to take into an account of the collective productivity of all workers and maximizing it opportunistically in response to stochastic changes in situational factors. This is also accompanied by a Multi-Agent Simulation to simulate the complex action and interactions of the drivers and passengers, and to analyze the effects of change in factors. After the implementation of the algorithm and simulation, we evaluated the effects in earnings, reputation and fatigue. In the most recent outbreak of the disease on COVID-19, the simulation also has a few mechanisms in showing how it spread among the drivers and passengers through the use of the proposed algorithm. 

## Getting started

```
go get github.com/harrizontal/dispatchserver
```

Please download the [map visualization tool](https://github.com/Harrizontal/Intelligent-Order-Matching-Sim)

## Dependencies

```
go get github.com/gorilla/websocket
go get github.com/paulmach/orb
go get github.com/pquerna/ffjson
go get github.com/starwander/goraph
go get gonum.org/v1/gonum/graph
go get golang.org/x/exp/rand
```


## Running the Dispatcher simulator server

```
go run github.com/harrizontal/dispatchserver/cmd/dsweb
```

After executing the command, you should get this:
```
[RoadNetwork - ProcessJSONFile]Current Working Directory: /Users/harrisonwjy/testgolang
[RoadNetwork - PopulateGraph]Graph populated - nodes:107861 links:121168
[RoadNetwork - PopulateGraph]Generating road network took 329.822171ms. Road network generated.
[RoadNetwork - GenerateQuadTree]Quadtree generated
[Server] Proceed to run simulation...
[Simulator]SendStats started 
[Simulator]SendMapData started 
```
Connect the simulator via the [map visualization tool](https://github.com/Harrizontal/Intelligent-Order-Matching-Sim). There are few instruction on how to run the simulation over there.

## Other non-essential files

### Preprocess ride sharing data
Before running the simulation, it is required to preprocess the ride data to reduce CPU and memory usage during the simulation run. An example of the preprocessed data located in assets/new_orders_first_1000.csv
```
go run github.com/harrizontal/dispatchserver/cmd/dsfilterdata
```


### Display the total nodes and links for road network

```
go run github.com/harrizontal/dispatchserver/cmd/dsroadnetwork
```