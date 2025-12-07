package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"cisco-mdt-generator/pkg/mdt_dialout"
	"cisco-mdt-generator/pkg/telemetry"
)

// BGPNeighbor represents a simulated BGP neighbor
type BGPNeighbor struct {
	Address       string
	RemoteAS      uint32
	State         string // "Established", "Idle", "Active", "Connect"
	StateCode     uint32 // 6=Established, 1=Idle, 3=Active, 2=Connect
	PrefixesRecv  uint32
	PrefixesSent  uint32
	Uptime        uint64 // seconds
	LastFlap      time.Time
	FlapCount     uint32
}

// EVPNState tracks EVPN route counts
type EVPNState struct {
	Type2Routes uint32 // MAC/IP routes
	Type3Routes uint32 // IMET routes
	Type5Routes uint32 // IP Prefix routes
	TotalRoutes uint32
}

// VNIState tracks per-VNI state
type VNIState struct {
	VNIID       uint32
	State       string // "Up", "Down"
	StateCode   uint32 // 1=Up, 0=Down
	MACCount    uint32
	VTEPCount   uint32
	ARPCount    uint32
}

func main() {
	server := flag.String("server", "10.10.20.10:57500", "gRPC MDT collector address")
	nodeID := flag.String("node", "leaf-101", "Simulated NX-OS leaf node-id-str")
	interval := flag.Duration("interval", 5*time.Second, "Interval between telemetry updates")
	flapChance := flag.Float64("flap-chance", 0.02, "Chance of BGP neighbor flap per interval (0.0-1.0)")

	flag.Parse()

	log.Printf("Connecting to MDT collector at %s ...", *server)

	conn, err := grpc.NewClient(*server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial collector: %v", err)
	}
	defer conn.Close()

	client := mdt_dialout.NewGRPCMdtDialoutClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.MdtDialout(ctx)
	if err != nil {
		log.Fatalf("failed to open MdtDialout stream: %v", err)
	}

	log.Printf("MDT dial-out stream established. Sending telemetry every %s ...", interval.String())

	// Initialize simulated state
	startTime := time.Now()
	reqID := int64(rand.Int63())

	// VXLAN counters
	var ingressBytes uint64 = 1_000_000
	var egressBytes uint64 = 2_000_000

	// BGP Neighbors (spine switches)
	bgpNeighbors := []*BGPNeighbor{
		{Address: "10.0.0.1", RemoteAS: 65001, State: "Established", StateCode: 6, PrefixesRecv: 150, PrefixesSent: 50, Uptime: 0, FlapCount: 0, LastFlap: startTime},
		{Address: "10.0.0.2", RemoteAS: 65001, State: "Established", StateCode: 6, PrefixesRecv: 148, PrefixesSent: 50, Uptime: 0, FlapCount: 0, LastFlap: startTime},
		{Address: "10.0.0.3", RemoteAS: 65002, State: "Established", StateCode: 6, PrefixesRecv: 145, PrefixesSent: 50, Uptime: 0, FlapCount: 0, LastFlap: startTime},
	}

	// EVPN state
	evpnState := &EVPNState{
		Type2Routes: 120,
		Type3Routes: 8,
		Type5Routes: 45,
		TotalRoutes: 173,
	}

	// VNI states
	vniStates := []*VNIState{
		{VNIID: 5000, State: "Up", StateCode: 1, MACCount: 45, VTEPCount: 3, ARPCount: 42},
		{VNIID: 5001, State: "Up", StateCode: 1, MACCount: 32, VTEPCount: 3, ARPCount: 30},
		{VNIID: 5002, State: "Up", StateCode: 1, MACCount: 28, VTEPCount: 3, ARPCount: 25},
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			// Update VXLAN counters
			ingressBytes += uint64(1000 + rand.Intn(5000))
			egressBytes += uint64(1500 + rand.Intn(5000))

			// Update BGP neighbor state (simulate occasional flaps)
			for _, neighbor := range bgpNeighbors {
				if neighbor.State == "Established" {
					neighbor.Uptime = uint64(now.Sub(neighbor.LastFlap).Seconds())
					// Random flap chance
					if rand.Float64() < *flapChance {
						neighbor.State = "Idle"
						neighbor.StateCode = 1
						neighbor.PrefixesRecv = 0
						neighbor.FlapCount++
						neighbor.LastFlap = now
						log.Printf("BGP neighbor %s FLAPPED to Idle (flap #%d)", neighbor.Address, neighbor.FlapCount)
					} else {
						// Small fluctuation in prefixes
						neighbor.PrefixesRecv = uint32(int(neighbor.PrefixesRecv) + rand.Intn(5) - 2)
					}
				} else {
					// Recover from flap after ~15-30 seconds
					if now.Sub(neighbor.LastFlap) > time.Duration(15+rand.Intn(15))*time.Second {
						neighbor.State = "Established"
						neighbor.StateCode = 6
						neighbor.PrefixesRecv = uint32(140 + rand.Intn(20))
						neighbor.LastFlap = now
						log.Printf("BGP neighbor %s RECOVERED to Established", neighbor.Address)
					}
				}
			}

			// Update EVPN route counts (small fluctuations)
			evpnState.Type2Routes = uint32(int(evpnState.Type2Routes) + rand.Intn(5) - 2)
			evpnState.Type3Routes = uint32(int(evpnState.Type3Routes) + rand.Intn(3) - 1)
			evpnState.Type5Routes = uint32(int(evpnState.Type5Routes) + rand.Intn(3) - 1)
			evpnState.TotalRoutes = evpnState.Type2Routes + evpnState.Type3Routes + evpnState.Type5Routes

			// Update VNI state (small MAC fluctuations)
			for _, vni := range vniStates {
				vni.MACCount = uint32(int(vni.MACCount) + rand.Intn(5) - 2)
				vni.ARPCount = uint32(int(vni.ARPCount) + rand.Intn(3) - 1)
			}

			// Send all telemetry messages
			messages := buildAllTelemetry(now, *nodeID, ingressBytes, egressBytes, bgpNeighbors, evpnState, vniStates)

			for _, telem := range messages {
				payload, err := telem.Marshal()
				if err != nil {
					log.Printf("failed to marshal Telemetry: %v", err)
					continue
				}

				msg := &mdt_dialout.MdtDialoutArgs{
					ReqId:  reqID,
					Data:   payload,
					Errors: "",
				}

				if err := stream.Send(msg); err != nil {
					log.Fatalf("failed to send MdtDialoutArgs: %v", err)
				}
			}

			log.Printf("Sent telemetry: vxlan=%d/%d, bgp_neighbors=%d, evpn_routes=%d, vnis=%d",
				ingressBytes, egressBytes, len(bgpNeighbors), evpnState.TotalRoutes, len(vniStates))
		}
	}
}

func buildAllTelemetry(t time.Time, nodeID string, ingressBytes, egressBytes uint64,
	bgpNeighbors []*BGPNeighbor, evpnState *EVPNState, vniStates []*VNIState) []*telemetry.Telemetry {

	var messages []*telemetry.Telemetry
	ts := uint64(t.UnixMilli())

	// 1. VXLAN interface stats
	messages = append(messages, buildVxlanTelemetry(ts, nodeID, 5000, ingressBytes, egressBytes))

	// 2. BGP neighbor telemetry
	messages = append(messages, buildBGPNeighborTelemetry(ts, nodeID, bgpNeighbors))

	// 3. EVPN route telemetry
	messages = append(messages, buildEVPNRouteTelemetry(ts, nodeID, evpnState))

	// 4. VNI state telemetry
	messages = append(messages, buildVNIStateTelemetry(ts, nodeID, vniStates))

	return messages
}

func buildVxlanTelemetry(ts uint64, nodeID string, vni uint32, ingressBytes, egressBytes uint64) *telemetry.Telemetry {
	row := telemetry.RowField(
		[]*telemetry.TelemetryField{
			telemetry.Uint32Field("vni-id", vni, ts),
			telemetry.StringField("name", "VNI-Leaf-Tenant-A", ts),
		},
		[]*telemetry.TelemetryField{
			telemetry.Uint64Field("ingress-bytes", ingressBytes, ts),
			telemetry.Uint64Field("egress-bytes", egressBytes, ts),
		},
		ts,
	)

	return &telemetry.Telemetry{
		NodeIDStr:           nodeID,
		SubscriptionIDStr:   "vxlan_stats",
		EncodingPath:        "Cisco-NX-OS-device:System/vxlan-items/inst-items",
		CollectionStartTime: ts,
		CollectionEndTime:   ts,
		MsgTimestamp:        ts,
		DataGpbkv:           []*telemetry.TelemetryField{row},
	}
}

func buildBGPNeighborTelemetry(ts uint64, nodeID string, neighbors []*BGPNeighbor) *telemetry.Telemetry {
	var rows []*telemetry.TelemetryField

	for _, n := range neighbors {
		row := telemetry.RowField(
			[]*telemetry.TelemetryField{
				telemetry.StringField("neighbor-address", n.Address, ts),
				telemetry.Uint32Field("remote-as", n.RemoteAS, ts),
			},
			[]*telemetry.TelemetryField{
				telemetry.StringField("state", n.State, ts),
				telemetry.Uint32Field("state-code", n.StateCode, ts),
				telemetry.Uint32Field("prefixes-received", n.PrefixesRecv, ts),
				telemetry.Uint32Field("prefixes-sent", n.PrefixesSent, ts),
				telemetry.Uint64Field("uptime-seconds", n.Uptime, ts),
				telemetry.Uint32Field("flap-count", n.FlapCount, ts),
			},
			ts,
		)
		rows = append(rows, row)
	}

	return &telemetry.Telemetry{
		NodeIDStr:           nodeID,
		SubscriptionIDStr:   "bgp_neighbors",
		EncodingPath:        "Cisco-NX-OS-device:System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list",
		CollectionStartTime: ts,
		CollectionEndTime:   ts,
		MsgTimestamp:        ts,
		DataGpbkv:           rows,
	}
}

func buildEVPNRouteTelemetry(ts uint64, nodeID string, evpn *EVPNState) *telemetry.Telemetry {
	row := telemetry.RowField(
		[]*telemetry.TelemetryField{
			telemetry.StringField("address-family", "l2vpn-evpn", ts),
		},
		[]*telemetry.TelemetryField{
			telemetry.Uint32Field("type2-routes", evpn.Type2Routes, ts),
			telemetry.Uint32Field("type3-routes", evpn.Type3Routes, ts),
			telemetry.Uint32Field("type5-routes", evpn.Type5Routes, ts),
			telemetry.Uint32Field("total-routes", evpn.TotalRoutes, ts),
		},
		ts,
	)

	return &telemetry.Telemetry{
		NodeIDStr:           nodeID,
		SubscriptionIDStr:   "evpn_routes",
		EncodingPath:        "Cisco-NX-OS-device:System/evpn-items/bdevi-items/BDEvi-list",
		CollectionStartTime: ts,
		CollectionEndTime:   ts,
		MsgTimestamp:        ts,
		DataGpbkv:           []*telemetry.TelemetryField{row},
	}
}

func buildVNIStateTelemetry(ts uint64, nodeID string, vniStates []*VNIState) *telemetry.Telemetry {
	var rows []*telemetry.TelemetryField

	for _, v := range vniStates {
		row := telemetry.RowField(
			[]*telemetry.TelemetryField{
				telemetry.Uint32Field("vni-id", v.VNIID, ts),
			},
			[]*telemetry.TelemetryField{
				telemetry.StringField("state", v.State, ts),
				telemetry.Uint32Field("state-code", v.StateCode, ts),
				telemetry.Uint32Field("mac-count", v.MACCount, ts),
				telemetry.Uint32Field("vtep-count", v.VTEPCount, ts),
				telemetry.Uint32Field("arp-count", v.ARPCount, ts),
			},
			ts,
		)
		rows = append(rows, row)
	}

	return &telemetry.Telemetry{
		NodeIDStr:           nodeID,
		SubscriptionIDStr:   "vni_state",
		EncodingPath:        "Cisco-NX-OS-device:System/eps-items/epId-items/Ep-list/nws-items/vni-items/Nw-list",
		CollectionStartTime: ts,
		CollectionEndTime:   ts,
		MsgTimestamp:        ts,
		DataGpbkv:           rows,
	}
}
