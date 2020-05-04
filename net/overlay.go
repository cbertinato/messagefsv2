package net

import (
	"fmt"
	"strings"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/kademlia"
)

const printedLength = 8

// Message ...
type Message struct {
	Contents string
}

// Network ...
type Network struct {
	Node    *noise.Node
	Overlay *kademlia.Protocol
}

// CreateNetwork instantiates the node on the network
func CreateNetwork(handler func(noise.HandlerContext) error) (*Network, error) {
	node, err := noise.NewNode()
	if err != nil {
		return nil, err
	}

	node.RegisterMessage(Message{}, UnmarshalMessage)

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
		return nil, err
	}

	network := &Network{
		Node:    node,
		Overlay: overlay,
	}

	return network, nil
}

// Marshal ...
func (m Message) Marshal() []byte {
	return []byte(m.Contents)
}

// UnmarshalMessage ...
func UnmarshalMessage(buf []byte) (Message, error) {
	return Message{Contents: strings.ToValidUTF8(string(buf), "")}, nil
}

// FmtPeers generates the given list of peers as a list of strings
func FmtPeers(ids []noise.ID) []string {
	var str []string
	for _, id := range ids {
		str = append(str, fmt.Sprintf("%s(%s)", id.Address, id.ID.String()[:printedLength]))
	}
	return str
}
