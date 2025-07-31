package config

const (
	// /opt/scenario-manager-api/data/ctfd_scenarios/<id>/scenario.zip
	ScenarioFolder = "data/ctfd_scenarios"
	// /opt/scenario-manager-api/data/ctfd_data/<id>/data.json
	CtfdDataFolder = "data/ctfd_data"
	// /opt/scenario-manager-api/data/topology_configs/<id>/topology.yml
	TopologyConfigFolder  = "data/topology_configs"
	DatabaseLocation      = "data/input/ludus.db"
	TimestampFormat       = "2006-01-02T15:04:05Z07:00"
	PoolFolder            = "data/pool"
	LudusAdminUrl         = "https://10.2.60.2:8081"
	LudusUrl              = "https://10.2.60.2:8080"
	MaxConcurrentRequests = 4
)
