package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ipfslite "github.com/adlrocha/ipfs-lite"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
)

func helpcmd() {
	fmt.Println(`[!] Commands available:
	* addFile_<file_dir>
	* add_<string>
	* get_<cid> 
	* connect_<peer_multiaddr>
	* call_<fxCid>_<argCid1>_<argCid2>
	* exit`)
}

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

// Process commands received from prompt
func processInput(ctx context.Context, ipfs *ipfslite.Peer, text string, done chan bool) error {
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, " ", "")
	words := strings.Split(text, "_")

	// Defer notifying the that processing is finished.
	defer func() {
		done <- true
	}()

	if words[0] == "exit" {
		os.Exit(0)
	}
	if words[0] == "help" {
		helpcmd()
		return nil
	}
	if len(words) < 2 {
		fmt.Println("Wrong number of arguments")
		return fmt.Errorf("Wrong number of arguments")
	}

	// If we use add we can add random content to the network.
	if words[0] == "addFile" {
		bytecode, err := ioutil.ReadFile(words[1])
		if err != nil {
			fmt.Println("Couldn't read file: ", err)
			return err
		}
		cid, err := ipfs.Deploy(ctx, bytecode)
		if err != nil {
			fmt.Println("Couldn't add file to IPFS: ", err)
			return err
		}
		fmt.Println("Added file with CID: ", cid)

	} else if words[0] == "add" {
		bytecode := []byte(words[1])
		cid, err := ipfs.Deploy(ctx, bytecode)
		if err != nil {
			fmt.Println("Couldn't add string to IPFS: ", err)
			return err
		}
		fmt.Println("Added string with CID: ", cid)
	} else if words[0] == "connect" {
		connectPeer(ctx, ipfs, words[1])

	} else if words[0] == "get" {
		c, err := cid.Decode(string(words[1]))
		if err != nil {
			fmt.Println("Couldn't parse CID: ", err)
		}
		rsc, err := ipfs.GetFile(ctx, c)
		d, err := ioutil.ReadAll(rsc)
		if err != nil {
			fmt.Println("Couldn't get file: ", err)
		}
		fmt.Println("Get: ", string(d))

	} else if words[0] == "call" {
		cids := []cid.Cid{}

		// Parse CIDs
		for _, cs := range words[1:] {
			c, err := cid.Decode(string(cs))
			if err != nil {
				fmt.Println("Couldn't parse CID: ", err)
			}
			cids = append(cids, c)
		}

		out, err := ipfs.Call(ctx, cids[0], cids[1:])
		if err != nil {
			fmt.Println("Couldn't run function: ", err)
		}
		fmt.Println("Output CID: ", out.String())

	} else {
		fmt.Println("[!] Wrong command")
		helpcmd()
	}

	return nil
}

// conectPeer connects to a peer in the network.
func connectPeer(ctx context.Context, ipfs *ipfslite.Peer, id string) error {
	maddr, err := ma.NewMultiaddr(id)
	if err != nil {
		fmt.Println("Invalid peer ID")
		return err
	}
	fmt.Println("Multiaddr", maddr)
	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		fmt.Println("Invalid peer info", err)
		return err
	}
	err = ipfs.ConnectPeer(*addrInfo)
	if err != nil {
		fmt.Println("Couldn't connect to peer", err)
		return err
	}
	fmt.Println("Connected successfully to peer")
	return nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("-- We are spinning up your IPFS node and your runtime -- ")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := spawnNode(ctx)

	ch := make(chan string)
	chSignal := make(chan os.Signal)
	done := make(chan bool)
	signal.Notify(chSignal, os.Interrupt, syscall.SIGTERM)

	// Prompt routine
	go func(ch chan string, done chan bool) {
		for {
			fmt.Print(">> Enter command: ")
			text, _ := reader.ReadString('\n')
			ch <- text
			<-done
		}
	}(ch, done)

	// Processing loop.
	for {
		select {
		case text := <-ch:
			processInput(ctx, p, text, done)

		case <-chSignal:
			fmt.Printf("\nUse exit to close the tool\n")
			fmt.Printf(">>  Enter command: ")

		}
	}
}
