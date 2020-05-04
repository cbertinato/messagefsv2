package net

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/kademlia"
)

type message struct {
	contents string
}

func (m message) Marshal() []byte {
	return []byte(m.contents)
}

func unmarshalMessage(buf []byte) (message, error) {
	return message{contents: strings.ToValidUTF8(string(buf), "")}, nil
}

func main() {
	node, err := noise.NewNode()
	if err != nil {
		panic(err)
	}
	defer node.Close()

	node.RegisterMessage(message{}, unmarshalMessage)

	events := kademlia.Events{
		OnPeerAdmitted: func(id noise.ID) {
			fmt.Printf("Learned about a new peer %s(%s).\n", id.Address, id.ID.String()[:printedLength])
		},
		OnPeerEvicted: func(id noise.ID) {
			fmt.Printf("Forgotten a peer %s(%s).\n", id.Address, id.ID.String()[:printedLength])
		},
	}

	overlay := kademlia.New(kademlia.WithProtocolEvents(events))

	// bind kademlia to the node
	node.Bind(overlay.Protocol())

	node.Handle(handle)

	if err := node.Listen(); err != nil {
		panic(err)
	}

	go getInput(node, overlay)

	// Wait until Ctrl+C or a termination call is done.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Close stdin to kill the input goroutine.
	if err := os.Stdin.Close(); err != nil {
		panic(err)
	}

	// Empty println.
	fmt.Println()
}

// handle incoming messages
func handle(ctx noise.HandlerContext) error {
	fmt.Printf("Got a message: '%s'\n", string(ctx.Data()))
	return nil
}

func fmtPeers(ids []noise.ID) []string {
	var str []string
	for _, id := range ids {
		str = append(str, fmt.Sprintf("%s(%s)", id.Address, id.ID.String()[:printedLength]))
	}
	return str
}

// discover peers
func discover(overlay *kademlia.Protocol) {
	ids := overlay.Discover()

	str := fmtPeers(ids)

	if len(ids) > 0 {
		fmt.Printf("Discovered %d peer(s): [%v]\n", len(ids), strings.Join(str, ", "))
	} else {
		fmt.Printf("No peers discovered.\n")
	}
}

// get input from stdin
func getInput(node *noise.Node, overlay *kademlia.Protocol) {
	r := bufio.NewReader(os.Stdin)

	for {
		buf, _, err := r.ReadLine()

		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			panic(err)
		}

		line := string(buf)
		if len(line) == 0 {
			continue
		}

		switch {
		case line == "/discover":
			discover(overlay)
			continue
		case line == "/peers":
			ids := overlay.Table().Peers()
			str := fmtPeers(ids)
			fmt.Printf("You know %d peer(s): [%v]\n", len(ids), strings.Join(str, ", "))
			continue
		case line == "/me":
			me := node.ID()
			fmt.Printf("%s(%s)\n", me.Address, me.ID.String()[:printedLength])
			continue
		case strings.Contains(line, "/ping"):
			addr := strings.Fields(line)[1]
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := node.Ping(ctx, addr)
			cancel()

			if err != nil {
				fmt.Printf("Failed to ping node (%s). Skipping... [error: %s]\n", addr, err)
			}
			continue
		default:
		}

		for _, id := range overlay.Table().Peers() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := node.SendMessage(ctx, id.Address, message{contents: line})
			cancel()

			if err != nil {
				fmt.Printf("Failed to send message to %s(%s). Skipping... [error: %s]\n",
					id.Address,
					id.ID.String()[:printedLength],
					err,
				)
				continue
			}
		}
	}
}
