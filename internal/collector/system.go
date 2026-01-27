package collector

import (
	"fmt"
	"time"

	"github.com/anurag/saviour/pkg/metrics"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemCollector collects system-level metrics
type SystemCollector struct {
	agentName  string
	diskMounts []string
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector(agentName string, diskMounts []string) *SystemCollector {
	return &SystemCollector{
		agentName:  agentName,
		diskMounts: diskMounts,
	}
}

// Collect gathers all system metrics
func (c *SystemCollector) Collect() (*metrics.SystemMetrics, error) {
	m := &metrics.SystemMetrics{
		Timestamp: time.Now(),
		AgentName: c.agentName,
	}

	// Collect CPU metrics
	cpuMetrics, err := c.collectCPU()
	if err != nil {
		return nil, fmt.Errorf("failed to collect CPU metrics: %w", err)
	}
	m.CPU = cpuMetrics

	// Collect memory metrics
	memMetrics, err := c.collectMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to collect memory metrics: %w", err)
	}
	m.Memory = memMetrics

	// Collect disk metrics
	diskMetrics, err := c.collectDisk()
	if err != nil {
		return nil, fmt.Errorf("failed to collect disk metrics: %w", err)
	}
	m.Disk = diskMetrics

	// Collect network metrics
	netMetrics, err := c.collectNetwork()
	if err != nil {
		return nil, fmt.Errorf("failed to collect network metrics: %w", err)
	}
	m.Network = netMetrics

	// Collect system info
	sysInfo, err := c.collectSystemInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to collect system info: %w", err)
	}
	m.SystemInfo = sysInfo

	return m, nil
}

func (c *SystemCollector) collectCPU() (metrics.CPUMetrics, error) {
	var m metrics.CPUMetrics

	// Overall CPU usage
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return m, err
	}
	if len(percentages) > 0 {
		m.UsagePercent = percentages[0]
	}

	// Per-core usage
	perCore, err := cpu.Percent(time.Second, true)
	if err != nil {
		return m, err
	}
	m.PerCorePercent = perCore

	// Load average
	loadAvg, err := load.Avg()
	if err != nil {
		return m, err
	}
	m.LoadAvg1 = loadAvg.Load1
	m.LoadAvg5 = loadAvg.Load5
	m.LoadAvg15 = loadAvg.Load15

	return m, nil
}

func (c *SystemCollector) collectMemory() (metrics.MemoryMetrics, error) {
	var m metrics.MemoryMetrics

	// Virtual memory
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return m, err
	}
	m.Total = vmem.Total
	m.Available = vmem.Available
	m.Used = vmem.Used
	m.UsedPercent = vmem.UsedPercent

	// Swap memory
	swap, err := mem.SwapMemory()
	if err != nil {
		return m, err
	}
	m.SwapTotal = swap.Total
	m.SwapUsed = swap.Used
	m.SwapPercent = swap.UsedPercent

	return m, nil
}

func (c *SystemCollector) collectDisk() ([]metrics.DiskMetrics, error) {
	var diskMetrics []metrics.DiskMetrics

	// If no specific mounts configured, get all partitions
	mounts := c.diskMounts
	if len(mounts) == 0 {
		partitions, err := disk.Partitions(false)
		if err != nil {
			return nil, err
		}
		for _, p := range partitions {
			mounts = append(mounts, p.Mountpoint)
		}
	}

	// Collect metrics for each mount point
	for _, mount := range mounts {
		usage, err := disk.Usage(mount)
		if err != nil {
			// Skip mounts that can't be read
			continue
		}

		dm := metrics.DiskMetrics{
			MountPoint:  usage.Path,
			Device:      "", // Will be populated if needed
			FSType:      usage.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
			InodesTotal: usage.InodesTotal,
			InodesUsed:  usage.InodesUsed,
			InodesFree:  usage.InodesFree,
		}
		diskMetrics = append(diskMetrics, dm)
	}

	return diskMetrics, nil
}

func (c *SystemCollector) collectNetwork() (metrics.NetworkMetrics, error) {
	var m metrics.NetworkMetrics

	// Get network I/O counters
	counters, err := net.IOCounters(false)
	if err != nil {
		return m, err
	}

	// Aggregate all interfaces
	for _, counter := range counters {
		m.BytesSent += counter.BytesSent
		m.BytesRecv += counter.BytesRecv
		m.PacketsSent += counter.PacketsSent
		m.PacketsRecv += counter.PacketsRecv
		m.ErrorsIn += counter.Errin
		m.ErrorsOut += counter.Errout
		m.DropsIn += counter.Dropin
		m.DropsOut += counter.Dropout
	}

	return m, nil
}

func (c *SystemCollector) collectSystemInfo() (metrics.SystemInfo, error) {
	var m metrics.SystemInfo

	info, err := host.Info()
	if err != nil {
		return m, err
	}

	m.Hostname = info.Hostname
	m.OS = info.OS
	m.Platform = info.Platform
	m.PlatformVersion = info.PlatformVersion
	m.KernelVersion = info.KernelVersion
	m.Uptime = info.Uptime

	return m, nil
}
