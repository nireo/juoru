package juoru

import (
	"encoding/json"
	"math/rand"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
)

type Event struct {
	Type string
	Data map[string]string
}

type Node struct {
	ID    string
	Addr  string
	Data  map[string]string
	Peers map[string]string
	mutex sync.RWMutex
}

func NewNode(id, addr string) *Node {
	return &Node{
		ID:    id,
		Addr:  addr,
		Data:  make(map[string]string),
		Peers: make(map[string]string),
	}
}

func (n *Node) listen(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

	}
}

func (n *Node) handleNodeConnection(conn net.Conn) {
	defer conn.Close()

	var event Event
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&event); err != nil {
		log.Err(err)
		return
	}

	switch event.Type {
	case "gossip":
		n.handleGossipEvent(event.Data)
	case "join":
		n.handleJoin(event.Data)
	}
}

func (n *Node) handleGossipEvent(data map[string]string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for k, v := range data {
		if _, exists := n.Data[k]; !exists {
			n.Data[k] = v
		}
	}
}

func (n *Node) handleJoin(data map[string]string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for id, addr := range data {
		if id != n.ID && addr != n.Addr {
			n.Peers[id] = addr
		}
	}
}

func (n *Node) getRandomPeer() string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	peers := make([]string, 0, len(n.Peers))
	for _, addr := range n.Peers {
		peers = append(peers, addr)
	}
	return peers[rand.Intn(len(peers))]
}
