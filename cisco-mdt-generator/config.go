package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete YAML configuration structure
type Config struct {
	Simulation   SimulationConfig      `yaml:"simulation"`
	VXLAN        VXLANConfig           `yaml:"vxlan"`
	BGPNeighbors []BGPNeighborConfig   `yaml:"bgp_neighbors"`
	EVPN         EVPNConfig            `yaml:"evpn"`
	VNIStates    []VNIStateConfig      `yaml:"vni_states"`
}

// SimulationConfig contains simulation behavior parameters
type SimulationConfig struct {
	FlapRecoveryMin int            `yaml:"flap_recovery_min"`
	FlapRecoveryMax int            `yaml:"flap_recovery_max"`
	Counters        CountersConfig `yaml:"counters"`
}

// CountersConfig defines increment ranges for various counters
type CountersConfig struct {
	VXLANIngressMin      int `yaml:"vxlan_ingress_min"`
	VXLANIngressMax      int `yaml:"vxlan_ingress_max"`
	VXLANEgressMin       int `yaml:"vxlan_egress_min"`
	VXLANEgressMax       int `yaml:"vxlan_egress_max"`
	BGPPrefixFluctuation int `yaml:"bgp_prefix_fluctuation"`
	EVPNType2Fluctuation int `yaml:"evpn_type2_fluctuation"`
	EVPNType3Fluctuation int `yaml:"evpn_type3_fluctuation"`
	EVPNType5Fluctuation int `yaml:"evpn_type5_fluctuation"`
	VNIMACFluctuation    int `yaml:"vni_mac_fluctuation"`
	VNIARPFluctuation    int `yaml:"vni_arp_fluctuation"`
}

// VXLANConfig defines VXLAN initial state
type VXLANConfig struct {
	InitialIngressBytes uint64 `yaml:"initial_ingress_bytes"`
	InitialEgressBytes  uint64 `yaml:"initial_egress_bytes"`
	VNIID               uint32 `yaml:"vni_id"`
	InterfaceName       string `yaml:"interface_name"`
}

// BGPNeighborConfig defines a BGP neighbor's initial configuration
type BGPNeighborConfig struct {
	Address             string `yaml:"address"`
	RemoteAS            uint32 `yaml:"remote_as"`
	InitialPrefixesRecv uint32 `yaml:"initial_prefixes_recv"`
	InitialPrefixesSent uint32 `yaml:"initial_prefixes_sent"`
}

// EVPNConfig defines EVPN route initial state
type EVPNConfig struct {
	Type2Routes uint32 `yaml:"type2_routes"`
	Type3Routes uint32 `yaml:"type3_routes"`
	Type5Routes uint32 `yaml:"type5_routes"`
}

// VNIStateConfig defines a VNI's initial state
type VNIStateConfig struct {
	VNIID            uint32 `yaml:"vni_id"`
	InitialMACCount  uint32 `yaml:"initial_mac_count"`
	InitialVTEPCount uint32 `yaml:"initial_vtep_count"`
	InitialARPCount  uint32 `yaml:"initial_arp_count"`
}

// DefaultConfig returns the hardcoded default configuration
// This preserves backward compatibility when no config file exists
func DefaultConfig() *Config {
	return &Config{
		Simulation: SimulationConfig{
			FlapRecoveryMin: 15,
			FlapRecoveryMax: 30,
			Counters: CountersConfig{
				VXLANIngressMin:      1000,
				VXLANIngressMax:      5000,
				VXLANEgressMin:       1500,
				VXLANEgressMax:       5000,
				BGPPrefixFluctuation: 5,
				EVPNType2Fluctuation: 5,
				EVPNType3Fluctuation: 3,
				EVPNType5Fluctuation: 3,
				VNIMACFluctuation:    5,
				VNIARPFluctuation:    3,
			},
		},
		VXLAN: VXLANConfig{
			InitialIngressBytes: 1_000_000,
			InitialEgressBytes:  2_000_000,
			VNIID:               5000,
			InterfaceName:       "VNI-Leaf-Tenant-A",
		},
		BGPNeighbors: []BGPNeighborConfig{
			{Address: "10.0.0.1", RemoteAS: 65001, InitialPrefixesRecv: 150, InitialPrefixesSent: 50},
			{Address: "10.0.0.2", RemoteAS: 65001, InitialPrefixesRecv: 148, InitialPrefixesSent: 50},
			{Address: "10.0.0.3", RemoteAS: 65002, InitialPrefixesRecv: 145, InitialPrefixesSent: 50},
		},
		EVPN: EVPNConfig{
			Type2Routes: 120,
			Type3Routes: 8,
			Type5Routes: 45,
		},
		VNIStates: []VNIStateConfig{
			{VNIID: 5000, InitialMACCount: 45, InitialVTEPCount: 3, InitialARPCount: 42},
			{VNIID: 5001, InitialMACCount: 32, InitialVTEPCount: 3, InitialARPCount: 30},
			{VNIID: 5002, InitialMACCount: 28, InitialVTEPCount: 3, InitialARPCount: 25},
		},
	}
}

// LoadConfig loads configuration from YAML file with fallback to defaults
func LoadConfig(configPath string) (*Config, error) {
	// If config path is empty or file doesn't exist, return defaults
	if configPath == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - use defaults (backward compatible)
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with defaults, then overlay YAML values
	config := DefaultConfig()

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// validateConfig ensures configuration values are sensible
func validateConfig(cfg *Config) error {
	// Validate flap recovery times
	if cfg.Simulation.FlapRecoveryMin <= 0 || cfg.Simulation.FlapRecoveryMax <= 0 {
		return fmt.Errorf("flap recovery times must be positive")
	}
	if cfg.Simulation.FlapRecoveryMin > cfg.Simulation.FlapRecoveryMax {
		return fmt.Errorf("flap_recovery_min (%d) cannot be greater than flap_recovery_max (%d)",
			cfg.Simulation.FlapRecoveryMin, cfg.Simulation.FlapRecoveryMax)
	}

	// Validate counter ranges
	if cfg.Simulation.Counters.VXLANIngressMin < 0 || cfg.Simulation.Counters.VXLANIngressMax < 0 {
		return fmt.Errorf("VXLAN counter ranges must be non-negative")
	}
	if cfg.Simulation.Counters.VXLANIngressMin > cfg.Simulation.Counters.VXLANIngressMax {
		return fmt.Errorf("vxlan_ingress_min cannot be greater than vxlan_ingress_max")
	}
	if cfg.Simulation.Counters.VXLANEgressMin > cfg.Simulation.Counters.VXLANEgressMax {
		return fmt.Errorf("vxlan_egress_min cannot be greater than vxlan_egress_max")
	}

	// Validate BGP neighbors exist
	if len(cfg.BGPNeighbors) == 0 {
		return fmt.Errorf("at least one BGP neighbor must be configured")
	}

	// Validate VNI states exist
	if len(cfg.VNIStates) == 0 {
		return fmt.Errorf("at least one VNI state must be configured")
	}

	return nil
}

// initBGPNeighborsFromConfig creates runtime BGP neighbor structs from config
func initBGPNeighborsFromConfig(cfg *Config, startTime time.Time) []*BGPNeighbor {
	neighbors := make([]*BGPNeighbor, len(cfg.BGPNeighbors))

	for i, nc := range cfg.BGPNeighbors {
		neighbors[i] = &BGPNeighbor{
			Address:      nc.Address,
			RemoteAS:     nc.RemoteAS,
			State:        "Established",
			StateCode:    6,
			PrefixesRecv: nc.InitialPrefixesRecv,
			PrefixesSent: nc.InitialPrefixesSent,
			Uptime:       0,
			FlapCount:    0,
			LastFlap:     startTime,
		}
	}

	return neighbors
}

// initEVPNStateFromConfig creates runtime EVPN state from config
func initEVPNStateFromConfig(cfg *Config) *EVPNState {
	return &EVPNState{
		Type2Routes: cfg.EVPN.Type2Routes,
		Type3Routes: cfg.EVPN.Type3Routes,
		Type5Routes: cfg.EVPN.Type5Routes,
		TotalRoutes: cfg.EVPN.Type2Routes + cfg.EVPN.Type3Routes + cfg.EVPN.Type5Routes,
	}
}

// initVNIStatesFromConfig creates runtime VNI state structs from config
func initVNIStatesFromConfig(cfg *Config) []*VNIState {
	states := make([]*VNIState, len(cfg.VNIStates))

	for i, vc := range cfg.VNIStates {
		states[i] = &VNIState{
			VNIID:     vc.VNIID,
			State:     "Up",
			StateCode: 1,
			MACCount:  vc.InitialMACCount,
			VTEPCount: vc.InitialVTEPCount,
			ARPCount:  vc.InitialARPCount,
		}
	}

	return states
}
