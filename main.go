package main

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

const (
	serverURL = "localhost:8080"
	authToken = "happy-blue-falcon-cheese"
	minWait   = 1               // Minimum wait time in seconds
	maxWait   = 64              // Maximum wait time in seconds
	interval  = 1 * time.Minute // Cycle time for data collection
)

type Message struct {
	Hostname       string
	OS             string
	Uptime         string
	Shell          string
	CPU            string
	MemStats       string
	TotalDiskSpace string
	FreeDiskSpace  string
	UsedDiskSpace  string
	SystemArch     string
	AuthToken      string
	CPUPercentage  string
	RAMPercentage  string
}

var ws *websocket.Conn

func connectToServer() (*websocket.Conn, error) {
	// Attempt to connect to the server with exponential backoff
	var err error
	wait := minWait
	for {
		u := url.URL{Scheme: "ws", Host: serverURL, Path: "/"}
		header := make(http.Header)
		header.Add("Authorization", authToken)
		ws, _, err = websocket.DefaultDialer.Dial(u.String(), header)
		if err == nil {
			fmt.Printf("\rOK: %s", time.Now().String()[:36])
			return ws, nil
		}
		fmt.Printf("\rFAIL: %s. Retrying in %d seconds...", time.Now().String()[:36], wait)
		time.Sleep(time.Duration(wait) * time.Second)
		wait = int(math.Min(float64(wait*2), float64(maxWait)))
	}
}

func main() {
	var err error
	ws, err = connectToServer()
	if err != nil {
		fmt.Printf("\rSERVER OFFLINE: %s", time.Now().String()[:36])
		return
	}
	defer ws.Close()

	// Main loop for sending system usage data
	for {
		hostname := getHostname()
		osName := getOSName()
		// kernel := getKernelVersion()
		uptime := getUptime()
		shell := getShell()
		cpu := getCPUName()
		memStats := getMemStats()
		totalDiskSpace := getTotalDiskSpace()
		freeDiskSpace := getFreeDiskSpace()
		usedDiskSpace := getUsedDiskSpace()
		systemArch, _ := getSystemArch()
		cpuPercentage, cpuErr := getCPUPercentage()
		if cpuErr != nil {
			fmt.Println("Failed to get CPU percentage:", err)
			return
		}

		ramPercentage, ramErr := getRAMPercentage()
		if ramErr != nil {
			fmt.Println("Failed to get RAM percentage:", err)
			return
		}
		msg := Message{
			Hostname: hostname,
			OS:       osName,
			// Kernel:         kernel,
			Uptime:         uptime,
			Shell:          shell,
			CPU:            cpu,
			MemStats:       memStats,
			TotalDiskSpace: totalDiskSpace,
			FreeDiskSpace:  freeDiskSpace,
			UsedDiskSpace:  usedDiskSpace,
			SystemArch:     systemArch,
			AuthToken:      authToken,
			CPUPercentage:  cpuPercentage,
			RAMPercentage:  ramPercentage,
		}

		err := ws.WriteJSON(msg)
		if err != nil {
			fmt.Printf("\rSERVER OFFLINE: %s", time.Now().String()[:36])
			// Try to reconnect if sending message fails
			ws.Close()
			ws, err = connectToServer()
			if err != nil {
				fmt.Printf("\rFailed to reconnect to server: %v", err)
				return
			}
		}

		// Wait until the next collection time
		time.Sleep(interval)
	}
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func getOSName() string {
	return runtime.GOOS
}

// func getKernelVersion() string {
// 	var kernelVersion string
// 	var err error
// 	switch runtime.GOOS {
// 	case "windows":
// 		output, err := exec.Command("ver").Output()
// 		if err != nil {
// 			fmt.Printf("Error retrieving kernel version on Windows: %v", err)
// 			return ""
// 		}
// 		kernelVersion = string(output)
// 	case "linux", "darwin":
// 		output, err := exec.Command("uname", "-r").Output()
// 		if err != nil {
// 			fmt.Printf("Error retrieving kernel version on %s: %v", runtime.GOOS, err)
// 			return ""
// 		}
// 		kernelVersion = string(output)
// 	case "freebsd", "openbsd", "netbsd":
// 		kernelVersion, err = syscall.Sysctl("kern.version")
// 		if err != nil {
// 			fmt.Printf("Error retrieving kernel version on BSD: %v", err)
// 			return ""
// 		}
// 	default:
// 		fmt.Printf("Error: Kernel version retrieval not implemented for %s", runtime.GOOS)
// 		return ""
// 	}

// 	return strings.TrimSpace(kernelVersion)
// }

func getUptime() string {
	var uptime string
	switch runtime.GOOS {
	case "windows":
		output, err := exec.Command("net", "stats", "srv").Output()
		if err != nil {
			fmt.Printf("Error retrieving uptime on Windows: %v", err)
			return ""
		}
		outputStr := string(output)
		uptimeStart := strings.Index(outputStr, "Statistics since ") + 19
		uptimeEnd := strings.Index(outputStr[uptimeStart:], "\r\n")
		uptime = outputStr[uptimeStart : uptimeStart+uptimeEnd]
	case "linux":
		output, err := exec.Command("uptime").Output()
		if err != nil {
			fmt.Printf("Error retrieving uptime on Linux: %v", err)
			return ""
		}
		outputStr := string(output)
		uptimeStart := strings.Index(outputStr, "up ") + 3
		uptimeEnd := strings.Index(outputStr[uptimeStart:], ",")
		uptime = outputStr[uptimeStart : uptimeStart+uptimeEnd]
	case "darwin":
		output, err := exec.Command("uptime").Output()
		if err != nil {
			fmt.Printf("Error retrieving uptime on Darwin: %v", err)
			return ""
		}
		outputStr := string(output)
		uptimeStart := strings.Index(outputStr, "up ") + 3
		uptimeEnd := strings.Index(outputStr[uptimeStart:], ",")
		uptime = outputStr[uptimeStart : uptimeStart+uptimeEnd]
	case "freebsd", "openbsd", "netbsd":
		output, err := exec.Command("uptime").Output()
		if err != nil {
			fmt.Printf("Error retrieving uptime on BSD: %v", err)
			return ""
		}
		outputStr := string(output)
		uptimeStart := strings.Index(outputStr, "up ") + 3
		uptimeEnd := strings.Index(outputStr[uptimeStart:], ",")
		uptime = outputStr[uptimeStart : uptimeStart+uptimeEnd]
	default:
		fmt.Printf("Error: Uptime retrieval not implemented for %s", runtime.GOOS)
		return ""
	}

	return uptime
}

func getShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "Unknown"
	}
	elements := strings.Split(shell, "/")
	shellName := elements[len(elements)-1]
	return shellName
}

func getCPUName() string {
	return runtime.GOARCH
}

func getMemStats() string {
	switch runtime.GOOS {
	case "darwin":
		output, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			fmt.Printf("\n\rError retrieving memory info on Darwin: %v", err)
			return ""
		}
		outputStr := strings.TrimSpace(string(output))
		memSize, err := strconv.ParseUint(outputStr, 10, 64)
		if err != nil {
			fmt.Printf("\n\rError parsing memory info on Darwin: %v", err)
			return ""
		}
		return strconv.FormatUint(memSize/(1024*1024), 10) + "MB"
	case "freebsd", "openbsd", "netbsd":
		output, err := exec.Command("sysctl", "-n", "hw.physmem").Output()
		if err != nil {
			fmt.Printf("\n\rError retrieving memory info on BSD: %v", err)
			return ""
		}
		outputStr := strings.TrimSpace(string(output))
		memSize, err := strconv.ParseUint(outputStr, 10, 64)
		if err != nil {
			fmt.Printf("\n\rError parsing memory info on BSD: %v", err)
			return ""
		}
		return strconv.FormatUint(memSize/(1024*1024), 10) + "MB"
	case "linux":
		output, err := exec.Command("free", "-m").Output()
		if err != nil {
			fmt.Printf("\n\rError retrieving memory info on Linux: %v", err)
			return ""
		}
		outputStr := string(output)
		memIndex := strings.Index(outputStr, "Mem:")
		if memIndex == -1 {
			fmt.Println("Error parsing memory info on Linux")
			return ""
		}
		memLines := strings.Split(outputStr[memIndex:], "\n")[0]
		memFields := strings.Fields(memLines)
		if len(memFields) < 2 {
			fmt.Println("Error parsing memory info on Linux")
			return ""
		}
		totalRAM, err := strconv.ParseUint(memFields[1], 10, 64)
		if err != nil {
			fmt.Printf("\n\rError parsing memory info on Linux: %v", err)
			return ""
		}
		return strconv.FormatUint(totalRAM, 10) + "MB"
	case "windows":
		output, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize").Output()
		if err != nil {
			fmt.Printf("\n\rError retrieving memory info on Windows: %v", err)
			return "Unknown"
		}
		outputStr := strings.TrimSpace(string(output))
		memorySize, err := strconv.ParseUint(outputStr, 10, 64)
		if err != nil {
			fmt.Printf("\n\rError parsing memory size on Windows: %v", err)
			return "Unknown"
		}
		return strconv.FormatUint(memorySize/1024, 10) + "MB"
	default:
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		totalRAM := mem.TotalAlloc / (1024 * 1024)
		return strconv.FormatUint(totalRAM, 10) + "MB"
	}
}
func getTotalDiskSpace() string {
	var output []byte
	var err error

	switch runtime.GOOS {
	case "windows":
		output, err = exec.Command("wmic", "logicaldisk", "where", "drivetype=3", "get", "size,freespace").Output()
	case "darwin", "linux":
		output, err = exec.Command("df", "-h", "-t").Output()
	default:
		fmt.Printf("\n\rerror: Disk usage retrieval not implemented for %s", runtime.GOOS)
		return "0"
	}

	if err != nil {
		fmt.Printf("\n\rerror retrieving disk usage: %v", err)
		return "0"
	}

	outputStr := strings.TrimSpace(string(output))
	var totalSize uint64

	if runtime.GOOS == "windows" {
		lines := strings.Split(outputStr, "\r\n")[1:]
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				size, _ := strconv.ParseUint(fields[0], 10, 64)
				totalSize += size
			}
		}
		return fmt.Sprintf("%d", totalSize/(1024*1024*1024))
	} else {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 && strings.Contains(fields[0], "/dev/") && fields[8] == "/" {
				return strings.TrimRight(fields[1], "BGMi")
			}
		}
		return "0"
	}
}

func getFreeDiskSpace() string {
	var output []byte
	var err error

	switch runtime.GOOS {
	case "windows":
		output, err = exec.Command("wmic", "logicaldisk", "where", "drivetype=3", "get", "size,freespace").Output()
	case "darwin", "linux":
		output, err = exec.Command("df", "-h", "-t").Output()
	default:
		fmt.Printf("error: Disk usage retrieval not implemented for %s", runtime.GOOS)
		return "0"
	}

	if err != nil {
		fmt.Printf("error retrieving disk usage: %v", err)
		return "0"
	}

	outputStr := strings.TrimSpace(string(output))
	var totalFree uint64

	if runtime.GOOS == "windows" {
		lines := strings.Split(outputStr, "\r\n")[1:]
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				free, _ := strconv.ParseUint(fields[1], 10, 64)
				totalFree += free
			}
		}
		return fmt.Sprintf("%d", totalFree/(1024*1024*1024))
	} else {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 && strings.Contains(fields[0], "/dev/") && fields[8] == "/" {
				return strings.TrimRight(fields[3], "BGMi")
			}
		}
		return "0"
	}
}

func getUsedDiskSpace() string {
	var output []byte
	var err error

	switch runtime.GOOS {
	case "windows":
		output, err = exec.Command("wmic", "logicaldisk", "where", "drivetype=3", "get", "size,freespace").Output()
	case "darwin", "linux":
		output, err = exec.Command("df", "-h", "-t").Output()
	default:
		fmt.Printf("error: Disk usage retrieval not implemented for %s", runtime.GOOS)
		return "0"
	}

	if err != nil {
		fmt.Printf("error retrieving disk usage: %v", err)
		return "0"
	}

	outputStr := strings.TrimSpace(string(output))
	var totalSize, totalFree uint64

	if runtime.GOOS == "windows" {
		lines := strings.Split(outputStr, "\r\n")[1:]
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				size, _ := strconv.ParseUint(fields[0], 10, 64)
				free, _ := strconv.ParseUint(fields[1], 10, 64)
				totalSize += size
				totalFree += free
			}
		}
		return fmt.Sprintf("%d", (totalSize-totalFree)/(1024*1024*1024))
	} else {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 && strings.Contains(fields[0], "/dev/") && fields[8] == "/" {
				size, _ := strconv.ParseFloat(strings.TrimRight(fields[1], "BGMi"), 64)
				avail, _ := strconv.ParseFloat(strings.TrimRight(fields[3], "BGMi"), 64)
				used := size - avail
				return fmt.Sprintf("%.1f", used)
			}
		}
		return "0"
	}
}

func getSystemArch() (string, error) {
	arch := runtime.GOARCH
	if arch == "" {
		return "", fmt.Errorf("unable to determine system architecture")
	}
	return arch, nil
}

func getCPUPercentage() (string, error) {
	p, err := cpu.Percent(0, false)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%.2f%%", p[0]), nil
}

func getRAMPercentage() (string, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%.2f%%", vmStat.UsedPercent), nil
}
