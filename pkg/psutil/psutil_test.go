package psutil

import (
	"fmt"
	"testing"
)

func TestCpuInfo(t *testing.T) {
	//CPU Count
	cpuCount, _ := CPUCounts(true)
	fmt.Printf("CPU Count: %d\n", cpuCount)

	// Cpu Info
	cpuInfo, _ := CPUInfo()
	fmt.Printf("Num CPU: %d, cores: %d\n", len(cpuInfo), cpuInfo[0].Cores)
	cpuMhz := uint64(cpuInfo[0].Mhz)
	fmt.Printf("CPU Mhz: %d\n", cpuMhz)
	fmt.Printf("CPU Info: %#v\n", cpuInfo)
}

func TestMemInfo(t *testing.T) {
	//memInfo
	memInfo, _ := VirtualMemory()
	fmt.Printf("Total memory: %d\nAvailable memory: %d\n", memInfo.Total, memInfo.Free)
	fmt.Printf("Mem Info: %+v\n", memInfo)
}
