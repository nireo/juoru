package juoru

import (
	"net"
	"testing"
	"time"
)

func TestAddData(t *testing.T) {
	node := NewNode("test", "localhost:8000")
	node.AddData("key", "value")
	if node.Data["key"] != "value" {
		t.Errorf("failed to add data")
	}
}

func TestHandleGossip(t *testing.T) {
	node := NewNode("test", "localhost:8000")
	data := map[string]string{"key1": "value1", "key2": "value2"}

	node.handleGossipEvent(data)
	if len(node.Data) != 2 || node.Data["key1"] != "value1" || node.Data["key2"] != "value2" {
		t.Errorf("failed setting node data")
	}
}

func TestHandleJoin(t *testing.T) {
	node := NewNode("test", "localhost:8080")
	data := map[string]string{"peer1": "localhost:8001", "peer2": "localhost:8002"}
	node.handleJoin(data)
	if len(node.Peers) != 2 || node.Peers["peer1"] != "localhost:8001" || node.Peers["peer2"] != "localhost:8002" {
		t.Error("failed to update node peers")
	}
}

func TestGetRandomPeer(t *testing.T) {
	node := NewNode("test", "localhost:8000")
	node.Peers = map[string]string{"peer1": "localhost:8001", "peer2": "localhost:8002"}
	peer := node.getRandomPeer()
	if peer != "localhost:8001" && peer != "localhost:8002" {
		t.Error("unexpected peer address")
	}
}

func TestNodeStart(t *testing.T) {
	node := NewNode("test", "localhost:8000")
	err := node.Start()
	if err != nil {
		t.Errorf("Node.Start() failed: %v", err)
	}

	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		t.Errorf("Failed to connect to started node: %v", err)
	}
	conn.Close()
}

func TestNodeJoin(t *testing.T) {
	node1 := NewNode("node1", "localhost:8001")
	node2 := NewNode("node2", "localhost:8002")

	err := node1.Start()
	if err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	err = node2.Join("localhost:8001")
	if err != nil {
		t.Fatalf("Node2 failed to join: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if len(node1.Peers) != 1 || node1.Peers["node2"] != "localhost:8002" {
		t.Errorf("Node1 peers not updated correctly after join")
	}
}

func TestGossipPropagation(t *testing.T) {
	node1 := NewNode("node1", "localhost:8001")
	node2 := NewNode("node2", "localhost:8002")

	err := node1.Start()
	if err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	err = node2.Start()
	if err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	err = node2.Join("localhost:8001")
	if err != nil {
		t.Fatalf("Node2 failed to join: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	node1.AddData("key1", "value1")

	time.Sleep(2 * time.Second)
	if node2.Data["key1"] != "value1" {
		t.Errorf("Gossip failed to propagate data from node1 to node2")
	}
}
