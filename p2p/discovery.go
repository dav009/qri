package p2p

import (
	"context"
	"fmt"
	"time"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	discovery "gx/ipfs/QmRQ76P5dgvxTujhfPsCRAG83rC15jgb1G9bKLuomuC6dQ/go-libp2p/p2p/discovery"
)

const qriSupportKey = "qri-support"

// StartDiscovery initiates peer discovery, allocating a discovery
// services if one doesn't exist, then registering to be notified on peer discovery
func (n *QriNode) StartDiscovery() error {
	if n.Discovery == nil {
		service, err := discovery.NewMdnsService(context.Background(), n.Host, time.Second*5)
		if err != nil {
			return err
		}
		n.Discovery = service
	}

	// Registering will call n.HandlePeerFound when peers are discovered
	n.Discovery.RegisterNotifee(n)

	return nil
}

// HandlePeerFound
func (n *QriNode) HandlePeerFound(pinfo pstore.PeerInfo) {

	// TODO - this explicit connection seems to move exchange along quicker, but
	// seems like a bad idea. resolve.
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	// defer cancel()
	// if err := n.Host.Connect(ctx, pinfo); err != nil {
	// 	fmt.Println("Failed to connect to peer found by discovery: ", err)
	// 	return
	// }

	// n.Host.Peerstore().Addrs(p)
	// if addrs := n.QriPeers.Addrs(pinfo.ID); len(addrs) != 0 {
	// 	return
	// }

	// first check to see if we've seen this peer before
	if _, err := n.Host.Peerstore().Get(pinfo.ID, qriSupportKey); err == nil {
		return
	} else if supports, err := n.SupportsQriProtocol(pinfo); supports && err == nil {
		fmt.Println("checking qri support for peer: ", pinfo.ID.Pretty())

		if err := n.Host.Peerstore().Put(pinfo.ID, qriSupportKey, supports); err != nil {
			fmt.Println("errror setting qri support flag", err.Error())
			return
		}

		if err := n.AddQriPeer(pinfo); err != nil {
			fmt.Println(err.Error())
		}
	} else if err != nil {
		fmt.Println("error checking for qri support:", err.Error())
	}

	// fmt.Println("connected to peer: ", pinfo.ID.Pretty())
}

// SupportsQriProtocol checks to see if this peer supports the qri
// streaming protocol, returns
func (n *QriNode) SupportsQriProtocol(pinfo pstore.PeerInfo) (bool, error) {
	protos, err := n.Host.Peerstore().GetProtocols(pinfo.ID)

	if err == nil {
		// fmt.Printf("peer %s supports the following protocols:", pinfo.ID.String())
		for _, p := range protos {
			fmt.Println("\t", p)
			if p == string(QriProtocolId) {
				return true, nil
			}
		}
	}

	return false, err
}

func (n *QriNode) AddQriPeer(pinfo pstore.PeerInfo) error {
	// add this peer to our store
	n.QriPeers.AddAddrs(pinfo.ID, pinfo.Addrs, pstore.TempAddrTTL)

	if profile, _ := n.Repo.Peers().GetPeer(pinfo.ID); profile != nil {
		// we've already seen this peer
		return nil
	}

	// Get this repo's profile information
	profile, err := n.Repo.Profile()
	if err != nil {
		fmt.Println("error getting node profile info:", err)
		return err
	}

	res, err := n.SendMessage(pinfo, &Message{
		Type:    MtPeerInfo,
		Payload: profile,
	})
	if err != nil {
		fmt.Println("send profile message error:", err.Error())
		return err
	}

	if res.Phase == MpResponse {
		if err := n.handleProfileResponse(pinfo, res); err != nil {
			fmt.Println("profile response error", err.Error())
			return err
		}
	}

	// TODO - move dataset list exchange into a better place
	// Also, get a DHT or something
	res, err = n.SendMessage(pinfo, &Message{
		Type: MtDatasets,
		Payload: &DatasetsReqParams{
			Limit:  30,
			Offset: 0,
		},
	})
	if err != nil {
		fmt.Println("send message error", err.Error())
		return err
	}
	if res.Phase == MpResponse {
		if err := n.handleDatasetsResponse(pinfo, res); err != nil {
			fmt.Println("dataset response error:", err.Error())
		}
	}

	return nil
}
