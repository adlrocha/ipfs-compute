package ipfslite

import (
	"context"

	"github.com/bytecodealliance/wasmtime-go"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
)

// IPFS node struct. Wraps Wasmtime runtime.
type IPFS struct {
	Peer    *Peer
	Runtime *wasmtime.Store
}

// New starts a new
func SpawnWithRuntime(ctx context.Context) *IPFS {
	// Bootstrappers are using 1024 keys. See:
	// https://github.com/ipfs/infra/issues/378
	crypto.MinRsaKeyBits = 1024

	//FIXME: If we spawn more htan one peer in the same directory they
	// will share their datastore.
	ds, err := BadgerDatastore("datastore")
	if err != nil {
		panic(err)
	}
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		panic(err)
	}

	listen, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

	h, dht, err := SetupLibp2p(
		ctx,
		priv,
		nil,
		[]multiaddr.Multiaddr{listen},
		ds,
		Libp2pOptionsExtra...,
	)

	if err != nil {
		panic(err)
	}

	logger.Info("Starting IPFS node...")
	lite, err := New(ctx, ds, h, dht, nil)
	if err != nil {
		panic(err)
	}

	logger.Info("Bootstrapping node...")
	lite.Bootstrap(DefaultBootstrapPeers())

	logger.Info("Initializing WASM runtime...")
	return &IPFS{
		lite,
		wasmtime.NewStore(wasmtime.NewEngine()),
	}
}

// // Call a function from the network.
// func (n *IPFS) Call(ctx context.Context, fn cid.Cid, args []cid.Cid) (*cid.Cid, error) {
// 	cids := append([]cid.Cid{fn}, args...)
// 	data := make([][]byte, 0)

// 	// Get the required data from the network.
// 	for _, c := range cids {
// 		rsc, err := n.Peer.GetFile(ctx, c)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err != nil {
// 			return nil, err

// 		}
// 		defer rsc.Close()
// 		d, err := ioutil.ReadAll(rsc)
// 		if err != nil {
// 			return nil, err

// 		}
// 		data = append(data, d)
// 	}

// 	// Use the function code downloaded to run the function
// 	module, err := wasmtime.NewModule(n.Runtime.Engine, data[0])
// 	if err != nil {
// 		return nil, err
// 	}

// 	// TODO: Input in memory.
// 	// Call computation
// 	// ...
// 	// TODO: How to specify the number of arguments required.
// 	// Our `hello.wat` file imports one item, so we create that function
// 	// here.
// 	// In this case we have one that receives a function. Need to standardize how it works.
// 	item := wasmtime.WrapFunc(n.Runtime, func() string {
// 		fmt.Println(string(data[1]))
// 	})

// 	// Next up we instantiate a module which is where we link in all our
// 	// imports. We've got one import so we pass that in here.
// 	instance, err := wasmtime.NewInstance(n.Runtime, module, []*wasmtime.Extern{item.AsExtern()})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// After we've instantiated we can lookup our `run` function and call
// 	// it.
// 	run := instance.GetExport("run").Func()
// 	_, err = run.Call()
// 	check(err)
// }
