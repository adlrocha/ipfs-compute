package ipfslite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/fxamacker/cbor"
	"github.com/ipfs/go-cid"
)

// TODO: All of this data structures should be determined in IPLD.

// Type of the data. Expressed with a name and a codec to encode/decode.
type Type struct {
	Name  string
	Codec cid.Cid
}

// // Data defined by its type and where the content is stored.
// type Data struct {
// 	Type    cid.Cid
// 	Content cid.Cid
// }

// FxABI function interface.
type FxABI struct {
	Name     string
	Bytecode cid.Cid // We could add a type here if we want to support several runtimes.
	// Args     []Data
	Args []Type
}

func (f *FxABI) Encode() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := cbor.NewEncoder(buf, cbor.CanonicalEncOptions()).Encode(f)
	return buf, err
}

func (f *FxABI) Decode(r io.Reader) error {
	err := cbor.NewDecoder(r).Decode(&f)
	return err
}

// Deploy a function to the network.
func (p *Peer) Deploy(ctx context.Context, bytecode []byte, args []Type) (*cid.Cid, error) {
	// TODO: Add an IPLD DAG instead of chunking files directly.
	bytecodeCid, err := p.AddFile(ctx, bytes.NewReader(bytecode), &AddParams{})
	fmt.Println("Bytecode deployed at: ", bytecodeCid)
	if err != nil {
		return nil, err
	}
	abi := FxABI{
		Name:     "fx", // For now the only supported name is "fx" for all functions.
		Bytecode: bytecodeCid.Cid(),
		Args:     args,
	}
	b, err := abi.Encode()
	if err != nil {
		return nil, err
	}
	root, err := p.AddFile(ctx, b, &AddParams{})
	rootCid := root.Cid()
	fmt.Println("ABI deployed at: ", rootCid)

	return &rootCid, err
}

// Call a function deployed in the network
func (p *Peer) Call(ctx context.Context, fnCid cid.Cid, argsCid []cid.Cid) (*cid.Cid, error) {

	abi := FxABI{}
	// Get the manifest.
	rsc, err := p.GetFile(ctx, fnCid)
	if err != nil {
		return nil, err
	}
	defer rsc.Close()
	abi.Decode(rsc)
	if err != nil {
		return nil, err
	}

	cids := append([]cid.Cid{abi.Bytecode}, argsCid...)
	data := make([][]byte, 0)

	// Get the required data from the network.
	for _, c := range cids {
		rsc, err := p.GetFile(ctx, c)
		if err != nil {
			return nil, err
		}
		defer rsc.Close()
		d, err := ioutil.ReadAll(rsc)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}

	module, err := wasmtime.NewModule(p.runtime.Engine, data[0])

	instance, err := wasmtime.NewInstance(p.runtime, module, []*wasmtime.Extern{})
	if err != nil {
		return nil, err
	}

	call32 := func(f *wasmtime.Func, args ...interface{}) (int32, error) {
		ret, err := f.Call(args...)
		if err != nil {
			return 0, err
		}
		return ret.(int32), nil
	}

	// Instantiate memory
	memory := instance.GetExport("memory").Memory()
	// Alloc function
	alloc := instance.GetExport("alloc").Func()
	// Function to call
	fx := instance.GetExport(abi.Name).Func()

	linearInput := []byte{}
	argOffsets := make([]interface{}, 0)
	// Concatenate inputs linearly
	for _, k := range data[1:] {
		linearInput = append(linearInput, k...)
		// Offsets to get parameters inside WASM.
		argOffsets = append(argOffsets, len(k))
	}

	// Allocating extra 100 just in case.
	a, err := call32(alloc, len(linearInput)+100)
	if err != nil {
		fmt.Println("Error calling Wasm function")
		return nil, err
	}
	buf := memory.UnsafeData()

	// Allocate arguments in memory
	for i, k := range linearInput {
		buf[int(a)+i] = k
	}

	// Prepare arguments putting allocated pointer first
	args := append([]interface{}{a}, argOffsets...)

	b, err := call32(fx, args...)
	if err != nil {
		fmt.Println("Error calling Wasm function")
		return nil, err
	}

	// Add cid to the network.
	output, err := p.AddFile(ctx, bytes.NewReader(buf[a:a+b]), &AddParams{})
	outputCid := output.Cid()

	if err != nil {
		return nil, err
	}
	return &outputCid, nil

}
