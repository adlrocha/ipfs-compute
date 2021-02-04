package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	ipfslite "github.com/adlrocha/ipfs-lite"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
)

func spawnNode(ctx context.Context) *ipfslite.Peer {
	// Bootstrappers are using 1024 keys. See:
	// https://github.com/ipfs/infra/issues/378
	crypto.MinRsaKeyBits = 1024

	var ds datastore.Batching
	ds, err := ipfslite.BadgerDatastore("datastore")
	if err != nil {
		ds, err = ipfslite.BadgerDatastore("datastore-" + time.Now().String())
		if err != nil {
			panic(err)
		}
	}
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		panic(err)
	}

	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

	h, dht, err := ipfslite.SetupLibp2p(
		ctx,
		priv,
		nil,
		[]multiaddr.Multiaddr{listen},
		ds,
		ipfslite.Libp2pOptionsExtra...,
	)

	fmt.Println("Listening at: ", h.Addrs())
	for _, i := range h.Addrs() {
		a := strings.Split(i.String(), "/")
		if a[1] == "ip4" && a[2] == "127.0.0.1" && a[3] == "tcp" {
			fmt.Println("Connect from other peers using: ")
			fmt.Printf("connect_/ip4/127.0.0.1/tcp/%v/p2p/%s\n", a[4], h.ID().Pretty())
		}

	}
	if err != nil {
		panic(err)
	}

	lite, err := ipfslite.New(ctx, ds, h, dht, nil)
	if err != nil {
		panic(err)
	}

	lite.Bootstrap(ipfslite.DefaultBootstrapPeers())

	return lite
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := spawnNode(ctx)

	if len(os.Args) != 2 {
		fmt.Println("No script specified")
	}
	script := os.Args[1]
	p.RunScript(ctx, script)
}
