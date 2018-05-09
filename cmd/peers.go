package cmd

import (
	"encoding/json"
	"fmt"

	// "github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

// peersCmd represents the info command
var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "commands for working with peers",
	Annotations: map[string]string{
		"group": "network",
	},
}

var peersInfoCmd = &cobra.Command{
	Use:   "info",
	Short: `Get info on a qri peer`,
	Example: `  show info on a peer named "b5":
  $ qri peers info b5`,
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			ErrExit(fmt.Errorf("peer name is required"))
		}

		printInfo("searching for peer %s...", args[0])
		req, err := peerRequests(true)
		ExitIfErr(err)

		p := &core.PeerInfoParams{
			Peername: args[0],
		}

		res := &config.ProfilePod{}
		err = req.Info(p, res)
		ExitIfErr(err)

		data, err := json.MarshalIndent(res, "", "  ")
		ExitIfErr(err)

		printSuccess(string(data))
	},
}

var peersListCmd = &cobra.Command{
	Use:   "list",
	Short: "list known qri peers",
	Long:  `lists the peers your qri node has seen before`,
	Example: `  list qri peers:
  $ qri peers list`,
	Aliases: []string{"ls"},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		ntwk, err := cmd.Flags().GetString("network")
		ExitIfErr(err)
		showCached, err := cmd.Flags().GetBool("cached")
		ExitIfErr(err)

		// TODO - resurrect
		// outformat := cmd.Flag("format").Value.String()
		// if outformat != "" {
		// 	format, err := dataset.ParseDataFormatString(outformat)
		// 	if err != nil {
		// 		ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		// 	}
		// 	if format != dataset.JSONDataFormat {
		// 		ErrExit(fmt.Errorf("invalid data format. currently only json or plaintext are supported"))
		// 	}
		// }

		req, err := peerRequests(false)
		ExitIfErr(err)

		if ntwk == "ipfs" {
			limit := 200
			res := []string{}
			err := req.ConnectedIPFSPeers(&limit, &res)
			ExitIfErr(err)

			fmt.Println("")
			for i, p := range res {
				printSuccess("%d.\t%s", i+1, p)
			}
		} else {
			limit := 200
			online := []*config.ProfilePod{}
			err := req.ConnectedQriProfiles(&limit, &online)
			ExitIfErr(err)

			// if we don't have an RPC client, assume we're not connected
			if rpcClient == nil && !showCached {
				printInfo("qri not connected, listing cached peers")
				showCached = true
			}

			peers := []*config.ProfilePod{}
			if showCached {
				err = req.List(&core.ListParams{Limit: limit, Offset: 0}, &peers)
				ExitIfErr(err)

				// TODO - this is dumb us a map
				for _, peer := range peers {
					for _, olp := range online {
						if peer.ID == olp.ID {
							peer.Online = true
						}
					}
				}
			} else {
				peers = online
			}

			fmt.Println("")
			for i, peer := range peers {
				printPeerInfo(i, peer)
			}
		}
	},
}

var peersConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect directly to a peer ID",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		pr, err := peerRequests(false)
		ExitIfErr(err)

		res := &config.ProfilePod{}
		err = pr.ConnectToPeer(&args[0], res)
		ExitIfErr(err)

		printPeerInfo(0, res)
	},
}

var peersDisconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "explicitly close a connection to a peer",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	// peersListCmd.Flags().StringP("format", "f", "", "set output format [json]")
	peersListCmd.Flags().StringP("network", "n", "", "list peers from connected networks. currently only accepts \"ipfs\"")
	peersListCmd.Flags().BoolP("cached", "c", false, "show peers that aren't online, but previously seen")

	peersCmd.AddCommand(peersInfoCmd, peersListCmd, peersConnectCmd, peersDisconnectCmd)
	RootCmd.AddCommand(peersCmd)
}
