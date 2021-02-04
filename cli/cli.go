package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ipfslite "github.com/adlrocha/ipfs-lite"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
)

func helpcmd() {
	fmt.Println(`[!] Commands available:
	* script_<file_dir>
	* addFile_<file_dir>
	* add_<string>
	* get_<cid>
	* abi_<cid>
	* deploy_<bytecode>_<fn1>&<fn2>_<typeArg1>&<typeArg2>
	* connect_<peer_multiaddr>
	* call_<fxCid>_<fxname>_<argCid1>&<argCid2>
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

func checkArgs(args []string, l int) error {
	if len(args) != l {
		fmt.Println("Wrong number of arguments")
		return fmt.Errorf("Wrong number of arguments")
	}
	return nil
}

// // Process commands received from prompt
// func processInput(ctx context.Context, ipfs *ipfslite.Peer, text string, done chan bool) error {
// 	// TODO: Do not trim spaces for now.
// 	// text = strings.ReplaceAll(text, "\n", "")
// 	// text = strings.ReplaceAll(text, " ", "")
// 	words := strings.Split(text, "_")

// 	// Defer notifying the that processing is finished.
// 	defer func() {
// 		done <- true
// 	}()

// 	if words[0] == "exit" {
// 		os.Exit(0)
// 	}
// 	if words[0] == "help" {
// 		helpcmd()
// 		return nil
// 	}
// 	if len(words) < 2 {
// 		fmt.Println("Wrong number of arguments")
// 		return fmt.Errorf("Wrong number of arguments")
// 	}

// 	// If we use add we can add random content to the network.
// 	if words[0] == "addFile" {
// 		bytecode, err := ioutil.ReadFile(words[1])
// 		if err != nil {
// 			fmt.Println("Couldn't read file: ", err)
// 			return err
// 		}

// 		cid, err := ipfs.AddFile(ctx, bytes.NewReader(bytecode), nil)
// 		if err != nil {
// 			fmt.Println("Couldn't add file to IPFS: ", err)
// 			return err
// 		}
// 		fmt.Println("Added file with CID: ", cid)

// 	} else if words[0] == "add" {
// 		bytecode := []byte(words[1])
// 		cid, err := ipfs.AddFile(ctx, bytes.NewReader(bytecode), nil)
// 		if err != nil {
// 			fmt.Println("Couldn't add string to IPFS: ", err)
// 			return err
// 		}
// 		fmt.Println("Added string with CID: ", cid)
// 	} else if words[0] == "connect" {
// 		connectPeer(ctx, ipfs, words[1])

// 	} else if words[0] == "get" {
// 		c, err := cid.Decode(string(words[1]))
// 		if err != nil {
// 			fmt.Println("Couldn't parse CID: ", err)
// 			return err
// 		}
// 		rsc, err := ipfs.GetFile(ctx, c)
// 		d, err := ioutil.ReadAll(rsc)
// 		if err != nil {
// 			fmt.Println("Couldn't get file: ", err)
// 			return err
// 		}
// 		fmt.Println("Get: ", string(d))

// 	} else if words[0] == "abi" {

// 		c, err := cid.Decode(string(words[1]))
// 		if err != nil {
// 			fmt.Println("Couldn't parse CID: ", err)
// 			return err
// 		}
// 		rsc, err := ipfs.GetFile(ctx, c)
// 		d, err := ioutil.ReadAll(rsc)
// 		abi := ipfslite.FxABI{}
// 		err = json.Unmarshal(d, &abi)
// 		// TODO: Change according to the encoding used for ABIs, ideally IPLD
// 		// err = abi.Decode(rsc)
// 		if err != nil {
// 			fmt.Println("Couldn't decode ABI: ", err)
// 			return err
// 		}
// 		fmt.Println("ABI: ", abi)
// 	} else if words[0] == "deploy" {
// 		if e := checkArgs(words, 4); e != nil {
// 			return e
// 		}
// 		bytecode, err := ioutil.ReadFile(words[1])
// 		if err != nil {
// 			fmt.Println("Couldn't read file: ", err)
// 			return err
// 		}

// 		fxIn := strings.Split(words[2], "&")
// 		argsIn := strings.Split(words[3], "&")

// 		args := []ipfslite.Type{}
// 		for _, k := range argsIn {
// 			args = append(args, ipfslite.Type{Name: k})
// 		}

// 		cid, err := ipfs.Deploy(ctx, fxIn, bytecode, args)
// 		if err != nil {
// 			fmt.Println("Couldn't deploy to IPFS: ", err)
// 			return err
// 		}
// 		fmt.Println("Deployed function at: ", cid)

// 	} else if words[0] == "connect" {
// 		connectPeer(ctx, ipfs, words[1])

// 	} else if words[0] == "get" {
// 		c, err := cid.Decode(string(words[1]))
// 		if err != nil {
// 			fmt.Println("Couldn't parse CID: ", err)
// 			return err
// 		}
// 		rsc, err := ipfs.GetFile(ctx, c)
// 		d, err := ioutil.ReadAll(rsc)
// 		if err != nil {
// 			fmt.Println("Couldn't get file: ", err)
// 			return err
// 		}
// 		fmt.Println("Get: ", string(d))

// 	} else if words[0] == "call" {
// 		if e := checkArgs(words, 4); e != nil {
// 			return e
// 		}
// 		cids := []cid.Cid{}

// 		fnCid, err := cid.Decode(string(words[1]))
// 		if err != nil {
// 			fmt.Println("Couldn't parse CID: ", err)
// 			return err
// 		}

// 		argsIn := strings.Split(words[3], "&")
// 		// Parse CIDs
// 		for _, cs := range argsIn {
// 			c, err := cid.Decode(string(cs))
// 			if err != nil {
// 				fmt.Println("Couldn't parse CID: ", err)
// 				return err
// 			}
// 			cids = append(cids, c)
// 		}

// 		out, err := ipfs.Call(ctx, fnCid, words[2], cids)
// 		if err != nil {
// 			fmt.Println("Couldn't run function: ", err)
// 			return err
// 		}
// 		fmt.Println("Output CID: ", out.String())

// 	} else {
// 		fmt.Println("[!] Wrong command")
// 		helpcmd()
// 	}

// 	return nil
// }

// // conectPeer connects to a peer in the network.
// func connectPeer(ctx context.Context, ipfs *ipfslite.Peer, id string) error {
// 	maddr, err := ma.NewMultiaddr(id)
// 	if err != nil {
// 		fmt.Println("Invalid peer ID")
// 		return err
// 	}
// 	fmt.Println("Multiaddr", maddr)
// 	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
// 	if err != nil {
// 		fmt.Println("Invalid peer info", err)
// 		return err
// 	}
// 	err = ipfs.ConnectPeer(*addrInfo)
// 	if err != nil {
// 		fmt.Println("Couldn't connect to peer", err)
// 		return err
// 	}
// 	fmt.Println("Connected successfully to peer")
// 	return nil
// }

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
			p.ExecCmd(ctx, text, done)

		case <-chSignal:
			fmt.Printf("\nUse exit to close the tool\n")
			fmt.Printf(">>  Enter command: ")

		}
	}
}
