package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	ipfslite "github.com/adlrocha/ipfs-lite"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	dagcbor "github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/fluent"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
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

var linkBuilder = cidlink.LinkBuilder{cid.Prefix{
	Version:  1,    // Usually '1'.
	Codec:    0x71, // dag-cbor as per multicodec
	MhType:   0x15, // sha3-384 as per multihash
	MhLength: 48,   // sha3-384 hash has a 48-byte sum.
}}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := spawnPeer(ctx)

	storage := make(map[ipld.Link][]byte)
	store := func(ipld.LinkContext) (io.Writer, ipld.StoreCommitter, error) {
		buf := bytes.Buffer{}
		return &buf, func(lnk ipld.Link) error {
			storage[lnk] = buf.Bytes()
			return nil
		}, nil
	}
	loader := func(lnk ipld.Link, _ ipld.LinkContext) (io.Reader, error) {
		return bytes.NewReader(storage[lnk]), nil
	}

	eric := fluent.MustBuildMap(basicnode.Prototype.Any, 1, func(na fluent.MapAssembler) {
		na.AssembleEntry("name").AssignString("Eric Myhre")
	})
	ericLink, _ := linkBuilder.Build(ctx, ipld.LinkContext{}, eric, store)
	daniel := fluent.MustBuildMap(basicnode.Prototype.Any, 1, func(na fluent.MapAssembler) {
		na.AssembleEntry("name").AssignString("Daniel Martí")
	})
	danielLink, _ := linkBuilder.Build(ctx, ipld.LinkContext{}, daniel, store)
	people := fluent.MustBuildList(basicnode.Prototype.Any, 2, func(na fluent.ListAssembler) {
		na.AssembleValue().AssignLink(ericLink)
		na.AssembleValue().AssignLink(danielLink)
	})
	peopleLink, _ := linkBuilder.Build(ctx, ipld.LinkContext{}, people, store)

	nb := basicnode.Prototype.Any.NewBuilder()
	_ = peopleLink.Load(ctx, ipld.LinkContext{}, nb, loader)
	people2 := nb.Build()
	for itr := people2.ListIterator(); !itr.Done(); {
		_, value, _ := itr.Next()
		personLink, _ := value.AsLink()

		nb := basicnode.Prototype.Any.NewBuilder()
		_ = personLink.Load(ctx, ipld.LinkContext{}, nb, loader)
		person := nb.Build()

		name, _ := person.LookupByString("name")
		nameStr, _ := name.AsString()
		fmt.Println(nameStr)
	}

	buf := new(bytes.Buffer)
	dagcbor.Encoder(people2, buf)
	root, err := p.AddFile(ctx, buf, &ipfslite.AddParams{})
	if err != nil {
		panic(err)
	}
	fmt.Println(root.Cid())

}

// FxABI expresses the function interfaace.
type FxABI struct {
	fx   cid.Cid
	args []cid.Cid
}
