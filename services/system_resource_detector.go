package services

import (
	"runtime"
)

// SystemResourceDetector 系统资源检测器
type SystemResourceDetector struct{}

// NewSystemResourceDetector 创建系统资源检测器
func NewSystemResourceDetector() *SystemResourceDetector {
	return &SystemResourceDetector{}
}

// SystemResources 系统资源信息
type SystemResources struct {
	CPUCores           int   // CPU核心数
	AvailableRAM       int64 // 可用内存（MB）
	TotalRAM           int64 // 总内存（MB）
	RecommendedThreads int   // 推荐线程数
}

// DetectResources 检测系统资源
func (d *SystemResourceDetector) DetectResources() *SystemResources {
	// 获取CPU核心数
	cpuCores := runtime.NumCPU()

	// 获取内存信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 计算可用内存（MB）
	totalRAM := int64(memStats.Sys / 1024 / 1024)
	availableRAM := int64((memStats.Sys - memStats.Alloc) / 1024 / 1024)

	// 计算推荐线程数
	// 策略：CPU核心数 * 2，但不超过8个，至少2个
	recommendedThreads := cpuCores * 2
	if recommendedThreads > 8 {
		recommendedThreads = 8
	}
	if recommendedThreads < 2 {
		recommendedThreads = 2
	}

	return &SystemResources{
		CPUCores:           cpuCores,
		AvailableRAM:       availableRAM,
		TotalRAM:           totalRAM,
		RecommendedThreads: recommendedThreads,
	}
}

// GetRecommendedThreadCount 获取推荐的线程数
func (d *SystemResourceDetector) GetRecommendedThreadCount() int {
	resources := d.DetectResources()
	return resources.RecommendedThreads
}
