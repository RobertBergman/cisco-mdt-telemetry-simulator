# Cisco NX-OS MDT Telemetry Simulator

A complete test environment for Cisco Model-Driven Telemetry (MDT) using gRPC dial-out with GPB-KV encoding. This project simulates a Cisco NX-OS VXLAN leaf switch streaming telemetry data to a Telegraf/InfluxDB/Grafana stack.

## Features

- **Go-based MDT Generator** - Simulates NX-OS telemetry streams
- **VXLAN Interface Counters** - Ingress/egress byte counters with realistic traffic patterns
- **BGP Neighbor Simulation** - State changes, flapping, prefix counts
- **EVPN Route Telemetry** - Type-2 (MAC/IP), Type-3 (IMET), Type-5 (IP Prefix) route counts
- **VNI State Monitoring** - Per-VNI MAC counts, VTEP counts, ARP entries
- **Grafana Dashboards** - Pre-built dashboards with Flux queries
- **Alerting** - BGP neighbor down and flap detection alerts

## Architecture

```
┌─────────────────┐     gRPC      ┌───────────┐     InfluxDB    ┌───────────┐
│  MDT Generator  │──────────────▶│  Telegraf │───────────────▶│  InfluxDB │
│  (NX-OS Sim)    │   Port 57500  │           │    Line Proto   │           │
└─────────────────┘               └───────────┘                 └───────────┘
                                                                      │
                                                                      │ Flux
                                                                      ▼
                                                                ┌───────────┐
                                                                │  Grafana  │
                                                                │ Port 4001 │
                                                                └───────────┘
```

## Quick Start

```bash
# Clone the repository
git clone https://github.com/RobertBergman/cisco-mdt-telemetry-simulator.git
cd cisco-mdt-telemetry-simulator

# Start the stack
docker compose up -d

# View logs
docker compose logs -f mdt-generator
```

Access Grafana at http://localhost:4001 (admin/admin)

## Configuration

The MDT generator supports YAML-based configuration for customizing the simulated network topology and behavior without modifying code.

### Configuration File

Create or modify `config/generator.yaml` to customize:

- **BGP Neighbors**: Addresses, AS numbers, initial prefix counts
- **VNI States**: VNI IDs, MAC/VTEP/ARP counts
- **EVPN Routes**: Type-2, Type-3, Type-5 route counts
- **Simulation Parameters**: Flap recovery times, counter increment ranges
- **VXLAN Settings**: Initial byte counters, VNI ID, interface name

### Example Configuration

```yaml
# Simulation behavior
simulation:
  flap_recovery_min: 15
  flap_recovery_max: 30
  counters:
    vxlan_ingress_min: 1000
    vxlan_ingress_max: 5000
    bgp_prefix_fluctuation: 5

# BGP neighbors (typically spine switches)
bgp_neighbors:
  - address: "10.0.0.1"
    remote_as: 65001
    initial_prefixes_recv: 150
    initial_prefixes_sent: 50
  - address: "10.0.0.2"
    remote_as: 65001
    initial_prefixes_recv: 148
    initial_prefixes_sent: 50

# VNI states
vni_states:
  - vni_id: 5000
    initial_mac_count: 45
    initial_vtep_count: 3
    initial_arp_count: 42
```

See [`config/generator.yaml`](config/generator.yaml) for the complete example with all options documented.

### Configuration Precedence

The generator uses a layered configuration approach:

1. **Hardcoded Defaults** - Built-in fallback values (matches current behavior)
2. **YAML Config File** - Overrides defaults if `config/generator.yaml` exists
3. **CLI Flags** - Override both defaults and config file (deployment-specific)

This allows you to:
- Run without any config file (backward compatible)
- Define standard topologies in YAML
- Override specific settings via CLI for testing

### Customization Examples

#### Simulate a Larger Topology

```yaml
# config/generator.yaml
bgp_neighbors:
  - address: "10.0.0.1"
    remote_as: 65001
    initial_prefixes_recv: 200
    initial_prefixes_sent: 100
  - address: "10.0.0.2"
    remote_as: 65001
    initial_prefixes_recv: 200
    initial_prefixes_sent: 100
  - address: "10.0.0.3"
    remote_as: 65002
    initial_prefixes_recv: 180
    initial_prefixes_sent: 90
  - address: "10.0.0.4"
    remote_as: 65002
    initial_prefixes_recv: 180
    initial_prefixes_sent: 90

vni_states:
  - vni_id: 6000
    initial_mac_count: 100
    initial_vtep_count: 5
    initial_arp_count: 95
  - vni_id: 6001
    initial_mac_count: 80
    initial_vtep_count: 5
    initial_arp_count: 75
```

#### Increase Flap Recovery Time

```yaml
# config/generator.yaml
simulation:
  flap_recovery_min: 30  # 30-60 seconds instead of 15-30
  flap_recovery_max: 60
```

#### Simulate More Aggressive Traffic

```yaml
# config/generator.yaml
simulation:
  counters:
    vxlan_ingress_min: 5000
    vxlan_ingress_max: 20000
    vxlan_egress_min: 5000
    vxlan_egress_max: 20000
```

## Dashboards

### VXLAN Telemetry Dashboard
- Interface ingress/egress rates (bytes/second)
- Current traffic statistics

### EVPN/BGP Telemetry Dashboard
- BGP neighbor states with color-coded indicators
- BGP state history timeline
- Flap count and prefix statistics
- EVPN route counts by type
- VNI status table with MAC/ARP counts

## Telemetry Paths Simulated

| Sensor Path | Description |
|-------------|-------------|
| `System/vxlan-items/inst-items` | VXLAN interface counters |
| `System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list` | BGP neighbor state |
| `System/evpn-items/bdevi-items/BDEvi-list` | EVPN route summary |
| `System/eps-items/epId-items/Ep-list/nws-items/vni-items/Nw-list` | VNI state |

---

## NDFC Telemetry Template

Use this template in Nexus Dashboard Fabric Controller (NDFC) to configure equivalent telemetry on production switches.

### Template: `MDT_Telemetry_VXLAN_EVPN`

```
##template variables
COLLECTOR_IP=10.10.20.10
COLLECTOR_PORT=57500
SUBSCRIPTION_ID=100
SAMPLE_INTERVAL=5000

##template content

feature telemetry

telemetry
  destination-group 100
    ip address $COLLECTOR_IP port $COLLECTOR_PORT protocol gRPC encoding GPB
  sensor-group 100
    path System/vxlan-items/inst-items depth unbounded
  sensor-group 101
    path System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list depth unbounded
  sensor-group 102
    path System/evpn-items/bdevi-items/BDEvi-list depth unbounded
  sensor-group 103
    path System/eps-items/epId-items/Ep-list/nws-items/vni-items/Nw-list depth unbounded
  subscription $SUBSCRIPTION_ID
    dst-grp 100
    snsr-grp 100 sample-interval $SAMPLE_INTERVAL
    snsr-grp 101 sample-interval $SAMPLE_INTERVAL
    snsr-grp 102 sample-interval $SAMPLE_INTERVAL
    snsr-grp 103 sample-interval $SAMPLE_INTERVAL

##
```

### NDFC Policy Configuration

1. Navigate to **Operations > Templates**
2. Create new template with the content above
3. Apply to leaf switches in your VXLAN fabric
4. Set variables:
   - `COLLECTOR_IP`: Your Telegraf server IP
   - `COLLECTOR_PORT`: 57500 (default)
   - `SUBSCRIPTION_ID`: Unique ID per switch
   - `SAMPLE_INTERVAL`: 5000ms recommended

---

## Manual Switch Configuration

For engineers who want to manually configure a real NX-OS switch (9.3(x) or later):

### Step 1: Enable Telemetry Feature

```
configure terminal
feature telemetry
```

### Step 2: Configure Destination Group

```
telemetry
  destination-group 100
    ip address 10.10.20.10 port 57500 protocol gRPC encoding GPB
```

### Step 3: Configure Sensor Groups

#### VXLAN Interface Counters
```
telemetry
  sensor-group 100
    path System/vxlan-items/inst-items depth unbounded
```

#### BGP Neighbor State
```
telemetry
  sensor-group 101
    path System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list depth unbounded
```

#### EVPN Route Summary
```
telemetry
  sensor-group 102
    path System/evpn-items/bdevi-items/BDEvi-list depth unbounded
```

#### VNI State (NVE)
```
telemetry
  sensor-group 103
    path System/eps-items/epId-items/Ep-list/nws-items/vni-items/Nw-list depth unbounded
```

### Step 4: Create Subscription

```
telemetry
  subscription 100
    dst-grp 100
    snsr-grp 100 sample-interval 5000
    snsr-grp 101 sample-interval 5000
    snsr-grp 102 sample-interval 5000
    snsr-grp 103 sample-interval 5000
```

### Step 5: Verify Configuration

```
show telemetry transport
show telemetry data collector details
show telemetry control database destinations
show telemetry control database sensor-paths
show telemetry control database subscriptions
```

### Complete Configuration Example

```
! NX-OS 9.3(x) or later
!
feature telemetry
feature nv overlay
feature bgp
feature vn-segment-vlan-based
nv overlay evpn

telemetry
  destination-group 100
    ip address 10.10.20.10 port 57500 protocol gRPC encoding GPB
  sensor-group 100
    path System/vxlan-items/inst-items depth unbounded
  sensor-group 101
    path System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list depth unbounded
  sensor-group 102
    path System/evpn-items/bdevi-items/BDEvi-list depth unbounded
  sensor-group 103
    path System/eps-items/epId-items/Ep-list/nws-items/vni-items/Nw-list depth unbounded
  subscription 100
    dst-grp 100
    snsr-grp 100 sample-interval 5000
    snsr-grp 101 sample-interval 5000
    snsr-grp 102 sample-interval 5000
    snsr-grp 103 sample-interval 5000
```

---

## Telegraf Configuration

The Telegraf configuration uses the `cisco_telemetry_mdt` input plugin:

```toml
[[inputs.cisco_telemetry_mdt]]
  transport = "grpc"
  service_address = ":57500"

[[outputs.influxdb_v2]]
  urls = ["http://influxdb:8086"]
  token = "my-super-secret-token"
  organization = "netops"
  bucket = "telemetry"
```

---

## Generator Options

```bash
docker compose run mdt-generator --help

Options:
  -server string      gRPC MDT collector address (default "10.10.20.10:57500")
  -node string        Simulated NX-OS node-id-str (default "leaf-101")
  -interval duration  Interval between telemetry updates (default 5s)
  -flap-chance float  Chance of BGP neighbor flap per interval (default 0.02)
  -config string      Path to YAML configuration file (default "config/generator.yaml")
```

### CLI Flags vs Configuration File

**CLI flags** are for deployment-specific settings that change per environment:
- `--server`: Where to send telemetry (varies by deployment)
- `--node`: Device identifier (unique per simulator instance)
- `--interval`: Update frequency (for testing different rates)
- `--flap-chance`: BGP instability simulation (for testing alerting)

**Configuration file** is for topology and simulation data that's reusable:
- BGP neighbor definitions
- VNI states and counts
- EVPN route distributions
- Simulation behavior parameters

**Precedence**: CLI flags always override config file values. This lets you define a standard topology in YAML but override specific settings at runtime.

### Adjusting Flap Rate

```yaml
# docker-compose.yml
mdt-generator:
  command: >
    --server telegraf:57500
    --node leaf-101
    --interval 5s
    --flap-chance 0.1  # 10% chance of flap per interval
    --config /app/config/generator.yaml
```

### Using a Custom Configuration File

```yaml
# docker-compose.yml
mdt-generator:
  volumes:
    - ./config:/app/config:ro
    - ./my-custom-topology.yaml:/app/custom.yaml:ro  # Mount custom config
  command: >
    --server telegraf:57500
    --node leaf-101
    --config /app/custom.yaml  # Use custom config
```

---

## Alerting

Pre-configured Grafana alerts:

| Alert | Condition | Severity |
|-------|-----------|----------|
| BGP Neighbor Down | state_code < 6 (not Established) | Critical |
| BGP Flap Detected | flap_count increases | Warning |

Configure contact points in Grafana UI under **Alerting > Contact points**.

---

## Project Structure

```
├── cisco-mdt-generator/
│   ├── main.go                 # MDT generator with BGP/EVPN simulation
│   ├── config.go               # YAML configuration loader
│   ├── Dockerfile
│   ├── go.mod
│   └── pkg/
│       ├── telemetry/          # GPB-KV telemetry encoding
│       └── mdt_dialout/        # gRPC dial-out client
├── config/
│   ├── generator.yaml          # Generator topology configuration
│   ├── telegraf/
│   │   └── telegraf.conf       # Telegraf MDT input config
│   └── grafana/
│       ├── dashboards/         # Pre-built dashboards
│       └── provisioning/       # Datasources and alerting
├── docker-compose.yml
└── README.md
```

---

## Requirements

- Docker and Docker Compose
- For real switches: NX-OS 9.3(x) or later with telemetry license

## License

MIT

## References

- [Cisco NX-OS Programmability Guide - Telemetry](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/progammability/guide/b-cisco-nexus-9000-series-nx-os-programmability-guide-93x/b-cisco-nexus-9000-series-nx-os-programmability-guide-93x_chapter_0101001.html)
- [Telegraf Cisco MDT Input Plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/cisco_telemetry_mdt)
- [NDFC Telemetry Configuration](https://www.cisco.com/c/en/us/td/docs/dcn/ndfc/1213/configuration/fabric-controller/cisco-ndfc-fabric-controller-configuration-guide-1213.html)
