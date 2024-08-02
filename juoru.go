package juoru

import (
	"encoding/json"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Event struct {
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
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
			log.Err(err)
			continue
		}

		go n.handleNodeConnection(conn)
	}
}

func (n *Node) handleNodeConnection(conn net.Conn) {
	defer conn.Close()

	var event Event
	if err := json.NewDecoder(conn).Decode(&event); err != nil {
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

func (n *Node) AddData(key, value string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.Data[key] = value
}

func (n *Node) sendGossip(peerAddr string) {
	n.mutex.RLock()
	event := Event{
		Type: "gossip",
		Data: n.Data,
	}
	n.mutex.RUnlock()

	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		log.Err(err)
		return
	}
	defer conn.Close()

	err = json.NewEncoder(conn).Encode(event)
	if err != nil {
		log.Err(err)
	}
}

func (n *Node) gossip() {
	for {
		time.Sleep(750 * time.Millisecond)
		if len(n.Peers) > 0 {
			peer := n.getRandomPeer()
			n.sendGossip(peer)
		}
	}
}

func (n *Node) Start() error {
	ln, err := net.Listen("tcp", n.Addr)
	if err != nil {
		log.Err(err)
		return err
	}

	go n.listen(ln)
	go n.gossip()

	return nil
}

func (n *Node) Join(bootstrapAddr string) error {
	conn, err := net.Dial("tcp", bootstrapAddr)
	if err != nil {
		log.Err(err)
		return err
	}
	defer conn.Close()

	event := Event{
		Type: "join",
		Data: map[string]string{n.ID: n.Addr},
	}

	return json.NewEncoder(conn).Encode(event)
}
