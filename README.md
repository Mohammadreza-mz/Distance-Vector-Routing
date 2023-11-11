# Distance Vector Routing
Dynamic implementation of Distance Vector Algorithm and parallel simulation of the algorithm.

## Usage:
Find the shortest path and route network packets based on it. Send a message through your desired network! Change the network dynamically and watch how it affects routing your message.

## Implementation:
sockes are used to create real connections between each pair of connected nodes.
A thread is used for CLI to test the algorithm and modify the network, and it has access to nodes. In this way, we have simulated the network, and we can create a new node, edge, or disconnect an existing edge by entering the appropriate command. Each node is only aware of its neighbors, and they don't have access to the network structure.
Bellman-Ford algorithm is used for finding the shortest path in the DVR algorithm.
Because sockets listen for messages parallelly and the algorithm runs simultaneously, it creates critical sections. So, the implementation has Mutex and Waitgroup(Semaphore).

## Parameters:
`MaxResponseTime`: The maximum time to wait for a response. If no response is received, the algorithm assumes that the connection has a problem and removes it from the routing table.

`MaxNode`: The limit of number of nodes in the simulation. You can easily change that.

`MaxHop`: Maximum number of intermediate nodes to consider a path valid. It is used to prevent count to infinity problem.

`PortOffset`: The offset used for the port of the sockets. For node id, the port PortOffset+id is being used.

`TTL`: is the total time to live for messages. It can be increased when using more extensive networks.

## CLI Commands:
`addNode id neighbors_count`: add a new node to the network.
id: id for the new node.
neighbors_count: number of current neighbors of the new node. To facilitate building the network, you can add neighbors of the new node in this stage. if neighbors_count > 0 in each of the next neighbors_count lines provide an edge description in `id2 weight` format.

`deleteNode id`: delete the node id. To make the simulation more realistic, the socket connections of the deleted node will remain connected, but they will go unresponsive. It doesn't contribute to sending messages or DVR algorithm. Because of the smart implementation of the routing algorithm, the other sides of its edges notice this change automatically after a period and change their routing table :).

`addEdge id1 id2 weight`: create a new edge between id1 and id2 nodes with the given cost.

`editEdge id1 id2 weight`: change the cost of the connection between id1 and id2 to the given cost.

`showRoutingTable id`: show the routing table of the node id. It shows where to send the packet with the given destination.

`showDistanceVector id`: show the distance vector of node id. It illustrates the current cost of sending a message from node id to any other node.

`message "text" id1 id2`: send an arbitrary message from id1 to id2. This can also be used to find the path from id1 to id2. The message has a TTL field to prevent it from looping around the network.

`exit`: terminate the simulation


## How to run:
Run the following command in the terminal and enter the mentioned simulation commands based on your need.

```bash
go run main.go
```