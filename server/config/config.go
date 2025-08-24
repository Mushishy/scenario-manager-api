package config

const (
	// /opt/scenario-manager-api/data/scenarios/<id>/<scenario_name>.zip
	CtfdScenarioFolder = "data/scenarios"
	// /opt/scenario-manager-api/data/topologies/<id>/<topology_name>.yml
	TopologyConfigFolder = "data/topologies"
	// /opt/scenario-manager-api/data/pools/<id>/pool.json
	// /opt/scenario-manager-api/data/pools/<id>/ctfd_data.json
	PoolFolder = "data/pools"
	// /opt/ludus/ludus.db
	DatabaseLocation      = "data/input/ludus.db"
	TimestampFormat       = "2006-01-02T15:04:05Z07:00"
	LudusAdminUrl         = "https://10.2.60.2:8081"
	LudusUrl              = "https://10.2.60.2:8080"
	MaxConcurrentRequests = 4
)
