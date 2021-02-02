package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	ipfslite "github.com/adlrocha/ipfs-lite"
	"github.com/bytecodealliance/wasmtime-go"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := ipfslite.SpawnWithRuntime(ctx)
	wasm, err := wasmtime.Wat2Wasm(`
      (module
        (import "" "hello" (func $hello))
        (func (export "run")
          (call $hello))
      )
	`)

	// Put code in the network.
	root, err := n.Peer.AddFile(ctx, bytes.NewReader(wasm), &ipfslite.AddParams{})
	check(err)
	fmt.Println("Put cid: ", root.Cid())

	// Get code from the network.
	rsc, err := n.Peer.GetFile(ctx, root.Cid())
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

	module, err := wasmtime.NewModule(n.Runtime.Engine, rcvWasm)
	check(err)

	// Our `hello.wat` file imports one item, so we create that function
	// here.
	item := wasmtime.WrapFunc(n.Runtime, func() {
		fmt.Println("Hello from WASM!")
	})

	// Next up we instantiate a module which is where we link in all our
	// imports. We've got one import so we pass that in here.
	instance, err := wasmtime.NewInstance(n.Runtime, module, []*wasmtime.Extern{item.AsExtern()})
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
