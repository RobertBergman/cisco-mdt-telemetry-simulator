package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
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
	configPath := flag.String("config", "config/generator.yaml", "Path to YAML configuration file")

	flag.Parse()

	// Load configuration with fallback to defaults
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if _, statErr := os.Stat(*configPath); statErr == nil {
		log.Printf("Loaded configuration from: %s", *configPath)
	} else {
		log.Printf("Config file not found, using hardcoded defaults")
	}

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

	// Initialize simulated state from configuration
	startTime := time.Now()
	reqID := int64(rand.Int63())

	// VXLAN counters from config
	ingressBytes := cfg.VXLAN.InitialIngressBytes
	egressBytes := cfg.VXLAN.InitialEgressBytes

	// BGP Neighbors from config
	bgpNeighbors := initBGPNeighborsFromConfig(cfg, startTime)

	// EVPN state from config
	evpnState := initEVPNStateFromConfig(cfg)

	// VNI states from config
	vniStates := initVNIStatesFromConfig(cfg)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			// Update VXLAN counters using config ranges
			ingressBytes += uint64(cfg.Simulation.Counters.VXLANIngressMin +
				rand.Intn(cfg.Simulation.Counters.VXLANIngressMax-cfg.Simulation.Counters.VXLANIngressMin))
			egressBytes += uint64(cfg.Simulation.Counters.VXLANEgressMin +
				rand.Intn(cfg.Simulation.Counters.VXLANEgressMax-cfg.Simulation.Counters.VXLANEgressMin))

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
						// Small fluctuation in prefixes using config
						fluctuation := cfg.Simulation.Counters.BGPPrefixFluctuation
						neighbor.PrefixesRecv = uint32(int(neighbor.PrefixesRecv) + rand.Intn(fluctuation*2+1) - fluctuation)
					}
				} else {
					// Recover from flap using config time range
					recoveryTime := time.Duration(cfg.Simulation.FlapRecoveryMin +
						rand.Intn(cfg.Simulation.FlapRecoveryMax-cfg.Simulation.FlapRecoveryMin)) * time.Second

					if now.Sub(neighbor.LastFlap) > recoveryTime {
						neighbor.State = "Established"
						neighbor.StateCode = 6
						neighbor.PrefixesRecv = uint32(140 + rand.Intn(20))
						neighbor.LastFlap = now
						log.Printf("BGP neighbor %s RECOVERED to Established", neighbor.Address)
					}
				}
			}

			// Update EVPN route counts using config fluctuations
			type2Fluct := cfg.Simulation.Counters.EVPNType2Fluctuation
			evpnState.Type2Routes = uint32(int(evpnState.Type2Routes) + rand.Intn(type2Fluct*2+1) - type2Fluct)

			type3Fluct := cfg.Simulation.Counters.EVPNType3Fluctuation
			evpnState.Type3Routes = uint32(int(evpnState.Type3Routes) + rand.Intn(type3Fluct*2+1) - type3Fluct)

			type5Fluct := cfg.Simulation.Counters.EVPNType5Fluctuation
			evpnState.Type5Routes = uint32(int(evpnState.Type5Routes) + rand.Intn(type5Fluct*2+1) - type5Fluct)

			evpnState.TotalRoutes = evpnState.Type2Routes + evpnState.Type3Routes + evpnState.Type5Routes

			// Update VNI state using config fluctuations
			for _, vni := range vniStates {
				macFluct := cfg.Simulation.Counters.VNIMACFluctuation
				vni.MACCount = uint32(int(vni.MACCount) + rand.Intn(macFluct*2+1) - macFluct)

				arpFluct := cfg.Simulation.Counters.VNIARPFluctuation
				vni.ARPCount = uint32(int(vni.ARPCount) + rand.Intn(arpFluct*2+1) - arpFluct)
			}

			// Send all telemetry messages
			messages := buildAllTelemetry(now, *nodeID, ingressBytes, egressBytes, bgpNeighbors, evpnState, vniStates, cfg)

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
	bgpNeighbors []*BGPNeighbor, evpnState *EVPNState, vniStates []*VNIState, cfg *Config) []*telemetry.Telemetry {

	var messages []*telemetry.Telemetry
	ts := uint64(t.UnixMilli())

	// 1. VXLAN interface stats using config values
	messages = append(messages, buildVxlanTelemetry(ts, nodeID, cfg.VXLAN.VNIID, cfg.VXLAN.InterfaceName, ingressBytes, egressBytes))

	// 2. BGP neighbor telemetry
	messages = append(messages, buildBGPNeighborTelemetry(ts, nodeID, bgpNeighbors))

	// 3. EVPN route telemetry
	messages = append(messages, buildEVPNRouteTelemetry(ts, nodeID, evpnState))

	// 4. VNI state telemetry
	messages = append(messages, buildVNIStateTelemetry(ts, nodeID, vniStates))

	return messages
}

func buildVxlanTelemetry(ts uint64, nodeID string, vni uint32, vniName string, ingressBytes, egressBytes uint64) *telemetry.Telemetry {
	row := telemetry.RowField(
		[]*telemetry.TelemetryField{
			telemetry.Uint32Field("vni-id", vni, ts),
			telemetry.StringField("name", vniName, ts),
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
