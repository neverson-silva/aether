package application

import (
	"aether/internal/modules/monitoring/domain"
)

func isActive(state string) bool {
	switch state {
	case "running", "restarting":
		return true
	default:
		return false
	}
}

func sumOwner(resources []domain.Resource, owner string) domain.Aggregate {
	var agg domain.Aggregate
	agg.Available = true
	for _, r := range resources {
		if r.Owner != owner {
			continue
		}
		agg.Count++
		// Storage is attributable even for stopped containers, unlike CPU/RAM.
		if r.Storage != nil {
			agg.StorageUsage += *r.Storage
		}
		if !r.Active {
			continue
		}
		agg.RunningCount++
		agg.CPUOfHost += r.CPUOfHost
		agg.MemUsage += r.MemUsage
		agg.NetRxRate += r.NetRxRate
		agg.NetTxRate += r.NetTxRate
	}
	return agg
}

// Aggregate splits the resources of a snapshot into Aether and User buckets
// and reconciles them against the host. The System/unaccounted bucket is the
// remainder after the attributed consumption is removed from the host, clamped
// at zero. Because container working-set memory and host used memory are
// measured differently (cache attribution), exact equality is not guaranteed;
// the UI must present these as approximate.
func Aggregate(resources []domain.Resource, host domain.Host) (aether, user, system domain.Aggregate) {
	aether = sumOwner(resources, domain.OwnerAether)
	user = sumOwner(resources, domain.OwnerUser)

	system = domain.Aggregate{Available: host.MemUsed > 0 || host.CPUPercent > 0}
	if host.MemUsed > aether.MemUsage+user.MemUsage {
		system.MemUsage = host.MemUsed - aether.MemUsage - user.MemUsage
	}
	systemCPU := host.CPUPercent - aether.CPUOfHost - user.CPUOfHost
	if systemCPU < 0 {
		systemCPU = 0
	}
	system.CPUOfHost = systemCPU
	if host.DiskUsed > aether.StorageUsage+user.StorageUsage {
		system.StorageUsage = host.DiskUsed - aether.StorageUsage - user.StorageUsage
	}
	if host.MemTotal > 0 {
		aether.MemPercent = float64(aether.MemUsage) / float64(host.MemTotal) * 100
		user.MemPercent = float64(user.MemUsage) / float64(host.MemTotal) * 100
		system.MemPercent = float64(system.MemUsage) / float64(host.MemTotal) * 100
	}
	return aether, user, system
}
