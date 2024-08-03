package juoru

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type DataEntry struct {
	Timestamp int64  `json:"timestamp"`
	Value     string `json:"value"`
}

type Event struct {
	Type string               `json:"type"`
	Data map[string]DataEntry `json:"data"`
}

type Node struct {
	ID        string
	Addr      string
	Data      map[string]DataEntry
	Peers     map[string]string
	mutex     sync.RWMutex
	MaxPeers  int
	listener  net.Listener
	closeChan chan struct{}
	wg        sync.WaitGroup
}

func NewNode(id, addr string, maxPeers int) *Node {
	return &Node{
		ID:        id,
		Addr:      addr,
		Data:      make(map[string]DataEntry),
		Peers:     make(map[string]string),
		MaxPeers:  maxPeers,
		closeChan: make(chan struct{}),
	}
}

func (n *Node) listen() {
	defer n.wg.Done()
	for {
		select {
		case <-n.closeChan:
			return
		default:
			conn, err := n.listener.Accept()
			if err != nil {
				select {
				case <-n.closeChan:
					return
				default:
					log.Err(err).Msg("could not accept connection")
					continue
				}
			}

			go n.handleNodeConnection(conn)
		}
	}
}

func (n *Node) Close() error {
	close(n.closeChan)

	if n.listener != nil {
		err := n.listener.Close()
		if err != nil {
			return fmt.Errorf("error closing listener: %v", err)
		}
	}

	// wait for everything to close
	n.wg.Wait()
	return nil
}

func (n *Node) handleNodeConnection(conn net.Conn) {
	defer conn.Close()

	connDone := make(chan struct{})
	go func() {
		select {
		case <-n.closeChan:
			conn.Close()
		case <-connDone:
		}
	}()

	var event Event
	if err := json.NewDecoder(conn).Decode(&event); err != nil {
		log.Err(err)
		return
	}

	close(connDone)

	switch event.Type {
	case "gossip":
		n.handleGossipEvent(event.Data)
	case "join":
		n.handleJoin(event.Data)
	}
}

func (n *Node) handleGossipEvent(data map[string]DataEntry) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for k, v := range data {
		if existing, ok := n.Data[k]; !ok || existing.Timestamp < v.Timestamp {
			n.Data[k] = v
		}
	}
}

func (n *Node) handleJoin(data map[string]DataEntry) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for id, d := range data {
		if id != n.ID && d.Value != n.Addr {
			n.AddPeer(id, d.Value)
		}
	}
}

func (n *Node) AddPeer(id, addr string) {
	if len(n.Peers) >= n.MaxPeers {
		// for looping the peers is pretty much random (not actually)
		for k := range n.Peers {
			delete(n.Peers, k)
			break
		}
	}

	n.Peers[id] = addr
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
	currentTime := time.Now().UnixNano()
	n.Data[key] = DataEntry{Value: value, Timestamp: currentTime}
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
		log.Err(err).Msg("failed to contact node for gossip")
		return
	}
	defer conn.Close()

	err = json.NewEncoder(conn).Encode(event)
	if err != nil {
		log.Err(err)
	}
}

func (n *Node) gossip() {
	defer n.wg.Done()

	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-n.closeChan:
			return
		case <-ticker.C:
			if len(n.Peers) > 0 {
				peer := n.getRandomPeer()
				n.sendGossip(peer)
			}
		}
	}
}

func (n *Node) Start() error {
	var err error
	n.listener, err = net.Listen("tcp", n.Addr)
	if err != nil {
		log.Err(err)
		return err
	}

	n.wg.Add(2)
	go n.listen()
	go n.gossip()

	return nil
}

func (n *Node) Join(bootstrapAddr string) error {
	conn, err := net.Dial("tcp", bootstrapAddr)
	if err != nil {
		log.Err(err).Msg("error dialing bootstrap")
		return err
	}
	defer conn.Close()

	event := Event{
		Type: "join",
		Data: map[string]DataEntry{n.ID: {Value: n.Addr, Timestamp: time.Now().Unix()}},
	}

	return json.NewEncoder(conn).Encode(event)
}

// func (n *Node) startAntiEntropy() {
// 	ticker := time.NewTicker(1 * time.Minute)
// 	go func() {
// 		for range ticker.C {
// 			n.doAntiEntropy()
// 		}
// 	}()
// }

// func (n *Node) doAntiEntropy() {
// 	peer := n.getRandomPeer()
// 	if peer == "" {
// 		log.Warn().Msg("could not get random peer for anti entropy")
// 		return
// 	}

// 	n.mutex.RLock()
// 	data := n.Data
// 	n.mutex.RUnlock()

// 	remote, err := n.fetchDataFromPeer(peer)
// 	if err != nil {
// 		log.Err(err).Msg("could not fetch data in anti entropy")
// 		return
// 	}

// 	n.mutex.Lock()
// 	defer n.mutex.Unlock()
// 	for k, v := range remote {

// 	}
// }

// func (n *Node) fetchDataFromPeer(peerAddr string) (map[string]string, error) {
// 	return nil, nil
// }
