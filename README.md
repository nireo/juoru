# juoru

A eventually consistent in-memory key-value store. The evetual consistency is guaranteed using the gossip protocol.

```go
node := juoru.NewNode("<uniqueid>", "localhost:9000", 50)
node.Join("localhost:9001") // Another node hosted here will get this.

// Broadcast data to the network
node.AddData("hello", "world")

// Read the nodes internal state
value := node.Data["hello"]
```
