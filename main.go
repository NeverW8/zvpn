package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const configDir = ".zvpn"
const lastConfigFile = ".last_config"
const logFile = "/tmp/zvpn.log"
const pidFile = "/tmp/zvpn.pid"

func main() {
	if os.Geteuid() != 0 {
		fmt.Println("This program must be run with sudo or as the root user.")
		return
	}

	if !isOpenVPNInstalled() {
		fmt.Println("OpenVPN is not installed on your system. Please install it first.")
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to get home directory:", err)
		return
	}

	configPath := filepath.Join(homeDir, configDir)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Configuration directory %s does not exist. Do you want to create it? (yes/no): ", configPath)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "yes" {
			if err := os.Mkdir(configPath, 0755); err != nil {
				fmt.Println("Failed to create configuration directory:", err)
				return
			}
			fmt.Println("Configuration directory created:", configPath)
		} else {
			fmt.Println("Aborting.")
			return
		}
	}

	if len(os.Args) < 2 {
		stopServiceIfNeeded()
		startWithPrompt(configPath)
	} else {
		switch os.Args[1] {
		case "--start":
			stopServiceIfNeeded()
			startLastUsedConfig(configPath)
		case "--stop":
			stopService()
		case "--status":
			showStatus()
		case "--log":
			showLog()
		default:
			fmt.Println("Unknown argument. Use --start, --stop, --status, or --log.")
		}
	}
}

func isOpenVPNInstalled() bool {
	_, err := exec.LookPath("openvpn")
	return err == nil
}

func startWithPrompt(configPath string) {
	files, err := os.ReadDir(configPath)
	if err != nil {
		fmt.Println("Failed to read configuration directory:", err)
		return
	}

	var configs []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".ovpn") {
			configs = append(configs, file.Name())
		}
	}

	if len(configs) == 0 {
		fmt.Println("No valid ovpn config files found in", configPath)
		return
	}

	fmt.Println("Select a configuration file to use:")
	for i, config := range configs {
		fmt.Printf("%d. %s\n", i+1, config)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter choice: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	index := 0

	fmt.Sscanf(choice, "%d", &index)
	if index < 1 || index > len(configs) {
		fmt.Println("Invalid choice")
		return
	}

	selectedConfig := configs[index-1]
	saveLastUsedConfig(configPath, selectedConfig)
	startService(filepath.Join(configPath, selectedConfig))
}

func saveLastUsedConfig(configPath, configName string) {
	err := os.WriteFile(filepath.Join(configPath, lastConfigFile), []byte(configName), 0644)
	if err != nil {
		fmt.Println("Failed to save last used configuration:", err)
	}
}

func startLastUsedConfig(configPath string) {
	lastConfig, err := os.ReadFile(filepath.Join(configPath, lastConfigFile))
	if err != nil {
		fmt.Println("Failed to read last used configuration:", err)
		return
	}

	startService(filepath.Join(configPath, strings.TrimSpace(string(lastConfig))))
}

func startService(config string) {
	cmd := exec.Command("sudo", "openvpn", "--config", config)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	logFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		fmt.Println("Failed to start service:", err)
		return
	}

	err = os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	if err != nil {
		fmt.Println("Failed to save PID:", err)
		return
	}

	fmt.Println("VPN started with configuration:", config)
}

func stopServiceIfNeeded() {
	if _, err := os.Stat(pidFile); err == nil {
		fmt.Println("An active VPN connection is detected. Stopping it before starting a new one.")
		stopService()
	}
}

func stopService() {
	cmd := exec.Command("sudo", "pkill", "openvpn")
	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to stop the VPN service:", err)
		return
	}

	if err := os.Remove(pidFile); err != nil {
		fmt.Println("Failed to remove PID file:", err)
	}

	fmt.Println("VPN service stopped.")
}

func showStatus() {
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("VPN service is not running.")
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		fmt.Println("Invalid PID in PID file.")
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("VPN service is not running.")
		return
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		fmt.Println("VPN service is not running.")
	} else {
		fmt.Println("VPN service is running.")
	}
}

func showLog() {
	logData, err := ioutil.ReadFile(logFile)
	if err != nil {
		fmt.Println("Failed to read log file:", err)
		return
	}
	fmt.Println("VPN Logs:")
	fmt.Println(string(logData))
}

