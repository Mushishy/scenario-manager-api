package utils

import "sync"

// Global deployment state manager
var (
	deployingPools  = make(map[string]bool) // Set of poolIds that are deploying
	deploymentMutex sync.RWMutex
)

// SetPoolDeploying sets a pool as deploying
func SetPoolDeploying(poolId string) {
	deploymentMutex.Lock()
	defer deploymentMutex.Unlock()
	deployingPools[poolId] = true
}

// IsPoolDeploying checks if a pool is currently deploying
func IsPoolDeploying(poolId string) bool {
	deploymentMutex.RLock()
	defer deploymentMutex.RUnlock()
	return deployingPools[poolId]
}

// ClearPoolDeploymentState removes the pool from deploying set
func ClearPoolDeploymentState(poolId string) {
	deploymentMutex.Lock()
	defer deploymentMutex.Unlock()
	delete(deployingPools, poolId)
}
