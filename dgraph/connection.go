package dgraph

import (
	"context"
	"log"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dg is the global Dgraph client instance
var Dg *dgo.Dgraph

// InitDgraph initializes the connection to Dgraph
func InitDgraph(address string) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Dgraph: %v", err)
	}

	dc := api.NewDgraphClient(conn)
	Dg = dgo.NewDgraphClient(dc)

	op := &api.Operation{
		Schema: `
			id: string @index(exact) .
			name: string .
			clock: string .
			depth: int .
			value: string .
			key: string .
			node: string .
			parent: [uid] .
			type Event {
				id
				name
				clock
				depth
				parent
				value
				key
				node
			}
		`,
	}

	if err := Dg.Alter(context.Background(), op); err != nil {
		log.Fatalf("Failed to set schema: %v", err)
	}

	log.Println("Connected to Dgraph and schema set successfully")
}
