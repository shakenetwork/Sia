package api

import (
	"errors"
	"testing"
	"time"

	"github.com/NebulousLabs/Sia/build"
	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/modules/gateway"
)

// TestGatewayStatus checks that the /gateway/status call is returning a corect
// peerlist.
func TestGatewayStatus(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	st, err := createServerTester(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer st.server.panicClose()

	var info GatewayGET
	st.getAPI("/gateway", &info)
	if len(info.Peers) != 0 {
		t.Fatal("/gateway gave bad peer list:", info.Peers)
	}
}

// TestGatewayPeerConnect checks that /gateway/connect is adding a peer to the
// gateway's peerlist.
func TestGatewayPeerConnect(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	st, err := createServerTester(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer st.server.panicClose()

	peer, err := gateway.New("localhost:0", false, build.TempDir("api", t.Name()+"2", "gateway"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := peer.Close()
		if err != nil {
			panic(err)
		}
	}()
	err = st.stdPostAPI("/gateway/connect/"+string(peer.Address()), nil)
	if err != nil {
		t.Fatal(err)
	}

	var info GatewayGET
	err = st.getAPI("/gateway", &info)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Peers) != 1 || info.Peers[0].NetAddress != peer.Address() {
		t.Fatal("/gateway/connect did not connect to peer", peer.Address())
	}
}

// TestGatewayPeerDisconnect checks that /gateway/disconnect removes the
// correct peer from the gateway's peerlist.
func TestGatewayPeerDisconnect(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	st, err := createServerTester(t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer st.server.panicClose()

	peer, err := gateway.New("localhost:0", false, build.TempDir("api", t.Name()+"2", "gateway"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := peer.Close()
		if err != nil {
			panic(err)
		}
	}()
	err = st.stdPostAPI("/gateway/connect/"+string(peer.Address()), nil)
	if err != nil {
		t.Fatal(err)
	}

	var info GatewayGET
	st.getAPI("/gateway", &info)
	if len(info.Peers) != 1 || info.Peers[0].NetAddress != peer.Address() {
		t.Fatal("/gateway/connect did not connect to peer", peer.Address())
	}

	err = st.stdPostAPI("/gateway/disconnect/"+string(peer.Address()), nil)
	if err != nil {
		t.Fatal(err)
	}
	err = st.getAPI("/gateway", &info)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Peers) != 0 {
		t.Fatal("/gateway/disconnect did not disconnect from peer", peer.Address())
	}
}

// TestGatewayOutboundPreference checks that the gateway always prefers to
// connect to outbound peers when it has a chance.
func TestGatewayOutboundPreference(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Create 6 peers, these peers will all be outbound for eachother.
	g1, err := createServerTester(t.Name() + "g1")
	if err != nil {
		t.Fatal(err)
	}
	// Mine an extra block with g1 so that the lead block is established.
	_, err = g1.miner.AddBlock()
	if err != nil {
		t.Fatal(err)
	}
	g2, err := createServerTester(t.Name() + "g2")
	if err != nil {
		t.Fatal(err)
	}
	g3, err := createServerTester(t.Name() + "g3")
	if err != nil {
		t.Fatal(err)
	}
	g4, err := createServerTester(t.Name() + "g4")
	if err != nil {
		t.Fatal(err)
	}
	g5, err := createServerTester(t.Name() + "g5")
	if err != nil {
		t.Fatal(err)
	}
	g6, err := createServerTester(t.Name() + "g6")
	if err != nil {
		t.Fatal(err)
	}
	outboundFriends := []*serverTester{g1, g2, g3, g4, g5, g6}
	err = fullyConnectNodes(outboundFriends)
	if err != nil {
		t.Fatal(err)
	}

	// Close one peer at a time, then wait until everyone has added the missing
	// peer as outbound. This means that all peers will be 'HasBeenOutbound' for
	// eachother in the outbound friends group, if everything is working
	// correctly.
	for i, g := range outboundFriends {
		// Close this g, wait until all the other g's have the full set of
		// outbound peers, then re-open this g.
		addr := g.gateway.Address()
		err := g.server.Close()
		if err != nil {
			t.Fatal(err)
		}
		err = retry(100, time.Millisecond*250, func() error {
			for j, h := range outboundFriends {
				// Don't check the peer we just closed.
				if j == i {
					continue
				}

				// Check that there are 4 outbound peers for each peer.
				var gg GatewayGET
				err := h.getAPI("/gateway", &gg)
				if err != nil {
					return err
				}
				numOutbound := 0
				for _, peer := range gg.Peers {
					if peer.Inbound {
						continue
					}
					numOutbound++
				}
				if numOutbound < 4 {
					return errors.New("expecting to have 4 outbound peers")
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(i, err)
		}

		// Re-open this g.
		outboundFriends[i], err = assembleGatewayServerTester(g.walletKey, g.dir, addr)
		if err != nil {
			t.Fatal(err)
		}
	}

	g7, err := createServerTester(t.Name() + "g7")
	if err != nil {
		t.Fatal(err)
	}
	g8, err := createServerTester(t.Name() + "g8")
	if err != nil {
		t.Fatal(err)
	}
	fullGroup := append(outboundFriends, g7, g8)

	// Connect the new peers to the outbound friends.
	for _, g := range outboundFriends {
		var gg GatewayGET
		err = g.getAPI("/gateway", &gg)
		if err != nil {
			t.Fatal(err)
		}
		err = g7.stdPostAPI("/gateway/connect/"+string(gg.NetAddress), nil)
		if err != nil && err.Error() != "already connected to this peer" {
			t.Fatal(err)
		}
		err = g8.stdPostAPI("/gateway/connect/"+string(gg.NetAddress), nil)
		if err != nil && err.Error() != "already connected to this peer" {
			t.Fatal(err)
		}
	}
	// Verify that everyone is connected and communicating.
	_, err = synchronizationCheck(fullGroup)
	if err != nil {
		t.Fatal(err)
	}

	// Grab the address of g7 and g8, so we can make sure we don't use them as
	// outbound peers in the new nodes.
	var g7addr, g8addr modules.NetAddress
	var gg GatewayGET
	err = g7.getAPI("/gateway", &gg)
	if err != nil {
		t.Fatal(err)
	}
	g7addr = gg.NetAddress
	err = g8.getAPI("/gateway", &gg)
	if err != nil {
		t.Fatal(err)
	}
	g8addr = gg.NetAddress
	println(g7addr)
	println(g8addr)

	// Close and re-open all of the outbound friends, one at a time. Upon
	// restart, they should only be connected to eachother, and not to the
	// newcomers.
	for i, g := range outboundFriends {
		// Close and reopen this g, resetting all of its connections.
		addr := g.gateway.Address()
		err := g.server.Close()
		if err != nil {
			t.Fatal(err)
		}
		outboundFriends[i], err = assembleGatewayServerTester(g.walletKey, g.dir, addr)
		if err != nil {
			t.Fatal(err)
		}

		// Block until all of the outbound friends have 4 outbound peers.
		err = retry(100, time.Millisecond*250, func() error {
			for _, h := range outboundFriends {
				// Get the peers.
				var gg GatewayGET
				err := h.getAPI("/gateway", &gg)
				if err != nil {
					return err
				}

				// Count the number of outbound peers. None should be g7addr or
				// g8addr, and there should be 4 total.
				numOutbound := 0
				for _, peer := range gg.Peers {
					if peer.Inbound {
						continue
					}
					numOutbound++

					if peer.NetAddress == g7addr || peer.NetAddress == g8addr {
						return errors.New("peer is not correctly using previous outbound peers as anchor nodes")
					}
				}
				if numOutbound < 4 {
					return errors.New("expecting to have 4 outbound peers")
				}
			}
			return nil
		})
		if err != nil {
			g.server.Close() // This will cause the most recent node list to be saved to disk.
			t.Fatal(i, err)
		}
	}

	// Close all of the outbound friends, and then re-open g1.
	g1 = outboundFriends[0]
	gAddr := g1.gateway.Address()
	err = g1.server.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = outboundFriends[0].server.Close()
	for _, g := range outboundFriends[1:] {
		err := g.server.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	g1, err = assembleGatewayServerTester(g1.walletKey, g1.dir, gAddr)
	if err != nil {
		t.Fatal(err)
	}

	// g1 should be able to connect to g7 and g8 as outbound peers.
	err = retry(100, time.Millisecond*250, func() error {
		// Get the peers.
		var gg GatewayGET
		err := g1.getAPI("/gateway", &gg)
		if err != nil {
			return err
		}

		// Count the number of outbound peers.
		numOutbound := 0
		for _, peer := range gg.Peers {
			if peer.Inbound {
				continue
			}
			numOutbound++
		}
		if numOutbound < 2 {
			return errors.New("expecting to have 4 outbound peers")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Close all open resources.
	err = g1.server.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = g8.server.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = g7.server.Close()
	if err != nil {
		t.Fatal(err)
	}
}
