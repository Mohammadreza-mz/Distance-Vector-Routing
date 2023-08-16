package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	INF             = 1e9
	MaxResponseTime = 5 * time.Second
	PortOffset      = 8000
	MaxNode         = 12
	MaxHop          = 16
)

type message struct {
	IsTable                        bool
	SenderId, ReceiverId, LastNode int
	TTL                            int
	DistanceVector                 [MaxNode]int
	Hop                            [MaxNode]int
	Text                           string
}

func writeToSocket(connection *net.Conn, m *message) {
	encoder := gob.NewEncoder(*connection)
	err := encoder.Encode(m)
	if err != nil {
		//fmt.Println(err)
		return
	}
}

func readFromSocket(connection *net.Conn, m *message) {
	dec := gob.NewDecoder(*connection)
	err := dec.Decode(m)
	if err != nil {
		//fmt.Println(err.Error())
		return
	}
}

func socketInputListener(connection *net.Conn, x *node, fromId int) {
	defer func() {
		if r := recover(); r != nil {
			if r := recover(); r != nil {
				fmt.Println("Recovered in listener", r)
			}
		}
	}()

	for {
		m := message{}
		readFromSocket(connection, &m)
		if x.isDeleted {
			return
		}
		if m.IsTable {
			ind := x.findNei(fromId)
			x.neighbors[ind].lastUpdate = time.Now()
			x.neighbors[ind].isNew = true
			for i := 0; i < MaxNode; i++ {
				x.neighbors[ind].latestDistance[i] = m.DistanceVector[i]
				x.neighbors[ind].hop[i] = m.Hop[i]
			}
		} else {
			if m.ReceiverId == x.id {
				fmt.Println("Received this message in node", x.id, " from node", m.SenderId, " :"+m.Text)
			} else if m.TTL > 0 {
				m.LastNode = x.id
				m.Text += " " + strconv.Itoa(x.id)
				if x.routingTable[m.ReceiverId] != -1 {
					nei := x.findNei(x.routingTable[m.ReceiverId])
					writeToSocket(x.neighbors[nei].connection, &m)
				}
			}
		}
	}
}

func createConnection(id1, id2 int) (con1, con2 *net.Conn) {
	wg := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}
	serverSocket := func(id int) {
		server, err := net.Listen("tcp", "localhost:"+strconv.Itoa(PortOffset+id))
		wg.Done()
		if err != nil {
			fmt.Println("Error listening:", err.Error())
			os.Exit(1)
		}
		defer server.Close()
		connection, err := server.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		con1 = &connection
		wg2.Done()
	}

	clientSocket := func(idToConnect int) {
		connection, err := net.Dial("tcp", "localhost:"+strconv.Itoa(PortOffset+idToConnect))
		if err != nil {
			panic(err)
		}
		con2 = &connection
	}
	wg.Add(1)
	wg2.Add(1)
	go serverSocket(id1)
	wg.Wait()
	clientSocket(id1)
	wg2.Wait()
	return
}

type neighbor struct {
	id, weight     int
	latestDistance [MaxNode]int
	hop            [MaxNode]int
	connection     *net.Conn
	lastUpdate     time.Time
	isNew          bool
}

type node struct {
	id           int
	neighbors    []neighbor
	neiMutex     sync.Mutex
	distance     [MaxNode]int
	hop          [MaxNode]int
	routingTable [MaxNode]int
	isDeleted    bool
}

type network struct {
	nodes []*node
}

func (n *network) findNode(id int) *node {
	for _, u := range n.nodes {
		if u.id == id {
			return u
		}
	}
	return nil
}

func (n *network) addNode() {
	var id, neighbors int
	fmt.Scan(&id, &neighbors)
	newNode := &node{id: id, isDeleted: false, neighbors: []neighbor{}}

	for i := 0; i < neighbors; i++ {
		var v, weight int
		fmt.Scan(&v, &weight)
		con1, con2 := createConnection(id, v)
		newNode.addNei(v, weight, con1)
		n.findNode(v).addNei(id, weight, con2)
	}

	for i := 0; i < MaxNode; i++ {
		newNode.distance[i] = INF
		newNode.routingTable[i] = -1
	}
	newNode.distance[id] = 0
	n.nodes = append(n.nodes, newNode)
	fmt.Println("added Node:", id)
	go DVR(newNode)
}

func (n *network) delNode(id int) {
	var ind int
	for i := 0; i < len(n.nodes); i++ {
		if n.nodes[i].id == id {
			ind = i
			n.nodes[i].isDeleted = true

			edges := []int{}
			for _, u := range n.nodes[i].neighbors {
				edges = append(edges, u.id)
			}
			for _, u := range edges {
				n.nodes[i].delNei(u)
			}
			break
		}
	}
	copy(n.nodes[ind:], n.nodes[ind+1:]) // Shift [ind+1:] left one index.
	n.nodes[len(n.nodes)-1] = &node{}    // Erase last element (write zero value).
	n.nodes = n.nodes[:len(n.nodes)-1]   // Truncate slice.
	fmt.Println("deleted Node:", id)
}

func (x *node) findNei(id int) int {
	for i, val := range x.neighbors {
		if val.id == id {
			return i
		}
	}
	return 0
}

func (x *node) delNei(id int) {
	x.neiMutex.Lock()
	ind := x.findNei(id)
	/*err := (*x.neighbors[ind].connection).Close()
	if err != nil {
		//it's okay to be unhandled
	}*/
	copy(x.neighbors[ind:], x.neighbors[ind+1:])   // Shift [ind+1:] left one index.
	x.neighbors[len(x.neighbors)-1] = neighbor{}   // Erase last element (write zero value).
	x.neighbors = x.neighbors[:len(x.neighbors)-1] // Truncate slice.

	x.neiMutex.Unlock()
	fmt.Println("deleted edge:", x.id, " ", id)
}

func (x *node) editNei(id, weight int) {
	x.neiMutex.Lock()
	ind := x.findNei(id)
	x.neighbors[ind].weight = weight
	x.neiMutex.Unlock()
	fmt.Println("edited edge:", x.id, " ", id, " ", weight)
}

func (x *node) addNei(id, weight int, connection *net.Conn) {
	x.neiMutex.Lock()
	nei := neighbor{id: id, weight: weight, isNew: true, lastUpdate: time.Now()}
	for i := 0; i < MaxNode; i++ {
		nei.latestDistance[i] = INF
	}
	//nei.latestDistance[id] = 0
	nei.connection = connection
	go socketInputListener(connection, x, id)
	x.neighbors = append(x.neighbors, nei)
	x.neiMutex.Unlock()
	fmt.Println("added edge:", x.id, " ", id, " ", weight)
}

func terminal(n *network) {
	for {
		var queryType string
		var id int
		fmt.Scan(&queryType)
		if queryType == "addNode" {
			n.addNode()
		} else if queryType == "delNode" {
			fmt.Scan(&id)
			n.delNode(id)
		} else if queryType == "addEdge" {
			var id1, id2, w int
			fmt.Scan(&id1, &id2, &w)
			node1 := n.findNode(id1)
			node2 := n.findNode(id2)
			con1, con2 := createConnection(id1, id2)
			node1.addNei(id2, w, con1)
			node2.addNei(id1, w, con2)
		} else if queryType == "delEdge" {
			var id1, id2 int
			fmt.Scan(&id1, &id2)
			node1 := n.findNode(id1)
			node1.delNei(id2)
			node2 := n.findNode(id2)
			node2.delNei(id1)
		} else if queryType == "editEdge" {
			var id1, id2, w int
			fmt.Scan(&id1, &id2, &w)
			node1 := n.findNode(id1)
			node1.editNei(id2, w)
			node2 := n.findNode(id2)
			node2.editNei(id1, w)
		} else if queryType == "showRoutingTable" {
			fmt.Scan(&id)
			node := n.findNode(id)
			fmt.Println("Routing Table from node ", id, ":")
			for i, val := range node.routingTable {
				if val != -1 {
					fmt.Print(i, ":", val, " ")
				}
			}
			fmt.Println()
		} else if queryType == "showDistanceVector" {
			fmt.Scan(&id)
			node := n.findNode(id)
			fmt.Println("distance vector from node ", id, ":")
			for i, val := range node.distance {
				if val != INF {
					fmt.Print(i, ":", val, " ")
				}
			}
			fmt.Println()
		} else if queryType == "exit" {
			for _, u := range n.nodes {
				n.delNode(u.id)
			}
			os.Exit(0)
		} else if queryType == "message" {
			var s string
			var id1, id2 int
			fmt.Scan(&s, &id1, &id2)
			m := &message{IsTable: false, TTL: 20, SenderId: id1, LastNode: id1, ReceiverId: id2, Text: s + " path:"}
			x := n.findNode(id1)
			if x.routingTable[id2] != -1 {
				nei := x.findNei(x.routingTable[id2])
				writeToSocket(x.neighbors[nei].connection, m)
			}
		} else {
			fmt.Println("invalid query name!")
		}
	}
}

// DVR runs Distance vector routing algorithm by using bellman ford
func DVR(x *node) {
	last := time.Now()
InfiniteLoop:
	for {
		disconnected := make(map[int]int)
		for last.Add(MaxResponseTime / 10).After(time.Now()) {

		}
		for {
			cnt := 0
			for _, u := range x.neighbors {
				if u.isNew {
					cnt++
					delete(disconnected, u.id)
				} else if u.lastUpdate.Add(MaxResponseTime).Before(time.Now()) {
					if disconnected[u.id] > 5 {
						delete(disconnected, u.id)
						//deleting a node from neighbors:
						x.delNei(u.id)
					} else {
						disconnected[u.id]++
						cnt++
					}
				}
			}
			if cnt == len(x.neighbors) {
				for ind, v := range x.neighbors {
					if disconnected[v.id] == 0 {
						x.neighbors[ind].isNew = false
					}
				}
				break
			}
		}

		for i := 1; i < MaxNode; i++ {
			x.distance[i] = INF
			x.routingTable[i] = -1
			x.hop[i] = MaxHop
			if i == x.id {
				x.distance[i] = 0
				x.hop[i] = 0
				continue
			}
			for _, v := range x.neighbors {
				if disconnected[v.id] != 0 {
					continue
				}
				if x.distance[i] > v.weight+v.latestDistance[i] && v.hop[i] < MaxHop {
					x.distance[i] = v.weight + v.latestDistance[i]
					x.routingTable[i] = v.id
					x.hop[i] = v.hop[i] + 1
				}
			}
		}

		for _, v := range x.neighbors {
			m := message{IsTable: true, SenderId: x.id, LastNode: x.id, TTL: 1}
			for i := 0; i < MaxNode; i++ {
				m.DistanceVector[i] = x.distance[i]
				m.Hop[i] = x.hop[i]
			}
			m.ReceiverId = v.id
			if x.isDeleted {
				break InfiniteLoop
			}
			writeToSocket(v.connection, &m)
		}
		last = time.Now()
	}

	return
	for _, v := range x.neighbors {
		err := (*v.connection).Close()
		if err != nil {
			//it's okay to be unhandled
		}
	}
}

func main() {
	n := &network{nodes: []*node{}}
	terminal(n)
}
