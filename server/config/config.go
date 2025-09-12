package config

const (
	/*
		Development constants
	*/
	TemplateCtfdTopologyLocation = "data/ctfd_topology.yml"
	CtfdScenarioFolder           = "data/scenarios"
	TopologyConfigFolder         = "data/topologies"
	PoolFolder                   = "data/pools"
	DatabaseLocation             = "data/input/ludus.db"

	/*
		Production constants
		CtfdScenarioFolder   = "/opt/scenario-manager-api/data/scenarios"
		TopologyConfigFolder = "/opt/scenario-manager-api/data/topologies"
		PoolFolder           = "/opt/scenario-manager-api/data/pools"
		DatabaseLocation     = "/opt/ludus/ludus.db"
	*/

	TimestampFormat       = "2006-01-02T15:04:05Z07:00"
	LudusAdminUrl         = "https://10.2.60.2:8081"
	LudusUrl              = "https://10.2.60.2:8080"
	MaxConcurrentRequests = 4
	ProxmoxNode           = "ludus"
	ProxmoxURL            = "https://10.2.60.2:8006"
	ProxmoxUsername       = "jan-slizik@pam"
	ProxmoxPassword       = "RDqTXYCfqkIYBG2kDGlM"
)
