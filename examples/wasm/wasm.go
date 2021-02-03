package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	ipfslite "github.com/adlrocha/ipfs-lite"
	"github.com/bytecodealliance/wasmtime-go"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multiaddr"
)

func spawnPeer(ctx context.Context) *ipfslite.Peer {
	// Bootstrappers are using 1024 keys. See:
	// https://github.com/ipfs/infra/issues/378
	crypto.MinRsaKeyBits = 1024

	//FIXME: If we spawn more than one peer in the same directory they
	// will share their datastore.
	ds, err := ipfslite.BadgerDatastore("datastore")
	if err != nil {
		panic(err)
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

func simple() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := spawnPeer(ctx)
	runtime := p.Runtime()
	wasm, err := wasmtime.Wat2Wasm(`
      (module
        (import "" "hello" (func $hello))
        (func (export "run")
          (call $hello))
      )
	`)

	// Put code in the network.
	root, err := p.AddFile(ctx, bytes.NewReader(wasm), &ipfslite.AddParams{})
	check(err)
	fmt.Println("Put cid: ", root.Cid())

	// Get code from the network.
	rsc, err := p.GetFile(ctx, root.Cid())
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	defer rsc.Close()
	rcvWasm, err := ioutil.ReadAll(rsc)
	if err != nil {
		panic(err)
	}

	module, err := wasmtime.NewModule(runtime.Engine, rcvWasm)
	check(err)

	// Our `hello.wat` file imports one item, so we create that function
	// here.
	item := wasmtime.WrapFunc(runtime, func() {
		fmt.Println("Hello from WASM!")
	})

	// Next up we instantiate a module which is where we link in all our
	// imports. We've got one import so we pass that in here.
	instance, err := wasmtime.NewInstance(runtime, module, []*wasmtime.Extern{item.AsExtern()})
	check(err)

	// After we've instantiated we can lookup our `run` function and call
	// it.
	run := instance.GetExport("run").Func()
	_, err = run.Call()
	check(err)

}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// End-to-end test of running an function
func mainSimple() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := spawnPeer(ctx)

	// Deploy code.
	bytecode, err := ioutil.ReadFile("../../functions/simple.wasm")
	// c, err := p.AddFile(ctx, bytes.NewReader(bytecode), nil)
	fnCid, err := p.Deploy(ctx, bytecode,
		[]ipfslite.Type{
			{Name: "string"},
		})
	check(err)

	// Add new string as an argument.
	argCid, err := p.AddFile(ctx, bytes.NewReader([]byte("Hello World!")), nil)
	check(err)

	// Call code
	output, err := p.Call(ctx, *fnCid, []cid.Cid{argCid.Cid()})
	check(err)
	fmt.Println("Output CID: ", output)
	// Get the manifest.
	rsc, err := p.GetFile(ctx, *output)
	check(err)
	defer rsc.Close()
	d, err := ioutil.ReadAll(rsc)
	check(err)
	fmt.Println("Result: ", string(d))

}

// End-to-end test of running an function
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := spawnPeer(ctx)

	// Deploy code.
	bytecode, err := ioutil.ReadFile("../../functions/wordcount.wasm")
	// c, err := p.AddFile(ctx, bytes.NewReader(bytecode), nil)
	fnCid, err := p.Deploy(ctx, bytecode,
		[]ipfslite.Type{
			{Name: "string"},
		})
	check(err)

	// Add new string as an argument.
	argCid, err := p.AddFile(ctx, bytes.NewReader([]byte("Hello World!")), nil)
	check(err)

	// Call code
	output, err := p.Call(ctx, *fnCid, []cid.Cid{argCid.Cid()})
	check(err)
	fmt.Println("Output CID: ", output)
	// Get the manifest.
	rsc, err := p.GetFile(ctx, *output)
	check(err)
	defer rsc.Close()
	d, err := ioutil.ReadAll(rsc)
	check(err)
	fmt.Println("Result: ", string(d))

}

// func testWasm() {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	p := spawnPeer(ctx)
// 	runtime := p.Runtime()

// 	// Deploy code.
// 	bytecode, err := ioutil.ReadFile("../../functions/simple.wasm")
// 	fnCid, err := p.Deploy(ctx, bytecode)
// 	check(err)

// 	// Deploy cid for argument
// 	argCid, err := p.Deploy(ctx, []byte("this is the arg!"))
// 	check(err)

// 	cids := append([]cid.Cid{fnCid}, argCid)
// 	data := make([][]byte, 0)

// 	// Get the required data from the network.
// 	for _, c := range cids {
// 		rsc, err := p.GetFile(ctx, c)
// 		check(err)
// 		defer rsc.Close()
// 		d, err := ioutil.ReadAll(rsc)
// 		check(err)
// 		data = append(data, d)
// 	}

// 	module, err := wasmtime.NewModule(runtime.Engine, data[0])

// 	instance, err := wasmtime.NewInstance(runtime, module, []*wasmtime.Extern{})
// 	check(err)

// 	call32 := func(f *wasmtime.Func, args ...interface{}) int32 {
// 		ret, err := f.Call(args...)
// 		check(err)
// 		return ret.(int32)
// 	}
// 	memory := instance.GetExport("memory").Memory()
// 	alloc := instance.GetExport("alloc").Func()
// 	append := instance.GetExport("append").Func()

// 	a := call32(alloc, 100)
// 	fmt.Println(memory.Size())
// 	fmt.Println(memory.DataSize())
// 	buf := memory.UnsafeData()

// 	fmt.Println("Alloc pointer: ", a)
// 	input := data[1]
// 	// Allocate in string
// 	for i, k := range input {
// 		buf[int(a)+i] = k
// 	}

// 	b := call32(append, a, len(input))
// 	fmt.Println("Result: ", a)
// 	fmt.Println("Output:", string(buf[a:a+b]))

// 	// Put in the CID

// 	// Deploy cid for argument
// 	outputCid, err := p.Deploy(ctx, buf[a:a+b])
// 	check(err)
// 	fmt.Println("Output CID: ", outputCid)

// 	// After we've instantiated we can lookup our `run` function and call
// 	// it.

// }
