package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Version is set at build time via ldflags
var version = "dev"

func printUsage() {
	fmt.Printf("Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Println("Commands:")
	fmt.Println("  run       Run transcribe mode - record and transcribe immediately")
	fmt.Println("  config    Show current configuration and config file location")
	fmt.Println("  download  Download a Whisper model")
	fmt.Println("  stop      Find and stop all running transcriber processes")
	fmt.Println("  version   Show version information")
	fmt.Println("  help      Show this help message")
	fmt.Println("\nOptions:")
	fmt.Println("  --output string")
	fmt.Println("        Output directory for transcriptions (default \".\")")
	fmt.Println("  --input string")
	fmt.Println("        Input directory for processing (defaults to temp directory)")
	fmt.Println("  --config string")
	fmt.Println("        Path to configuration file (defaults to standard config location ~/.transcriber/)")
	fmt.Println("  --duration string")
	fmt.Println("        Recording duration for run mode (e.g., 30s, 2m, 1h) (default \"30m\")")
	fmt.Println("  --model string")
	fmt.Println("        Model name to download (default \"ggml-large-v3-turbo-q5_0\")")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s run --output ./transcriptions\n", os.Args[0])
	fmt.Printf("  %s run --duration 2m --output ./transcriptions\n", os.Args[0])
	fmt.Printf("  %s config\n", os.Args[0])
	fmt.Printf("  %s download --model base\n", os.Args[0])
}

func printVersion() {
	fmt.Printf("%s version %s\n", filepath.Base(os.Args[0]), version)
	fmt.Printf("Built with %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func printProcessInfo() {
	pid := os.Getpid()
	fmt.Printf("\nProcess Information:\n")
	fmt.Printf("PID: %d\n", pid)

	// Show appropriate kill command based on OS
	if runtime.GOOS == "windows" {
		fmt.Printf("To gracefully exit: taskkill /PID %d /T\n", pid)
	} else {
		fmt.Printf("To gracefully exit: kill -TERM %d\n", pid)
	}
}

func killAllProcesses() error {
	var cmd *exec.Cmd
	var err error

	if runtime.GOOS == "windows" {
		// Windows: use tasklist and taskkill
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq transcriber*", "/FO", "CSV", "/NH")
	} else {
		// Unix-like: use ps and grep
		cmd = exec.Command("ps", "aux")
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list processes: %v", err)
	}

	var pids []string
	currentPID := os.Getpid()

	if runtime.GOOS == "windows" {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "transcriber") {
				fields := strings.Split(line, ",")
				if len(fields) >= 2 {
					pid := strings.Trim(fields[1], "\"")
					if pidInt, err := strconv.Atoi(pid); err == nil && pidInt != currentPID {
						pids = append(pids, pid)
					}
				}
			}
		}
	} else {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "transcriber") && !strings.Contains(line, "grep") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if pidInt, err := strconv.Atoi(fields[1]); err == nil && pidInt != currentPID {
						pids = append(pids, fields[1])
					}
				}
			}
		}
	}

	if len(pids) == 0 {
		fmt.Println("No other transcriber processes found running.")
		return nil
	}

	fmt.Printf("Found %d transcriber process(es) to kill: %v\n", len(pids), pids)

	for _, pid := range pids {
		var killCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			killCmd = exec.Command("taskkill", "/PID", pid, "/T", "/F")
		} else {
			killCmd = exec.Command("kill", "-TERM", pid)
		}

		if err := killCmd.Run(); err != nil {
			fmt.Printf("Failed to kill process %s: %v\n", pid, err)
		} else {
			fmt.Printf("Successfully killed process %s\n", pid)
		}
	}

	return nil
}

func getDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".transcriber"
	}
	return filepath.Join(homeDir, ".transcriber")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please specify a command")
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Check for help command early
	if command == "help" || command == "-help" || command == "--help" {
		printUsage()
		return
	}

	// Check for version command
	if command == "version" || command == "-version" || command == "--version" {
		printVersion()
		return
	}

	// Validate command
	validCommands := map[string]bool{
		"run":      true,
		"process":  true,
		"config":   true,
		"download": true,
		"stop":     true,
		"version":  true,
	}

	if !validCommands[command] {
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	// Parse flags for the subcommand
	flagSet := flag.NewFlagSet(command, flag.ExitOnError)
	var (
		outputDir  = flagSet.String("output", ".", "Output directory for transcriptions")
		configPath = flagSet.String("config", getDefaultConfigPath(), "Path to configuration file (defaults to ~/.transcriber/)")
		modelName  = flagSet.String("model", "ggml-large-v3-turbo-q5_0", "Model name to download")
	)

	flagSet.Usage = printUsage
	flagSet.Parse(os.Args[2:])

	transcriber, err := NewTranscriber(*configPath)
	if err != nil {
		fmt.Printf("Error initializing transcriber: %v\n", err)
		os.Exit(1)
	}

	switch command {

	case "run":
		printProcessInfo()
		if err := transcriber.RunTranscribe(*outputDir, true); err != nil {
			fmt.Printf("Error in run transcribe: %v\n", err)
			os.Exit(1)
		}

	case "config":
		config := transcriber.GetConfig()
		configJSON, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("Current configuration:\n%s\n\n", string(configJSON))
		fmt.Printf("Config file location: %s\n", transcriber.GetConfigPath())
		fmt.Println("To update configuration, edit the config file directly and restart the application.")

	case "download":
		// make configPath directory if it doesn't exist
		println("Config directory:", *configPath)
		if err := os.MkdirAll(*configPath, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			os.Exit(1)
		}
		if err := downloadModel(*modelName, *configPath); err != nil {
			fmt.Printf("Error downloading model: %v\n", err)
			os.Exit(1)
		}

		// Update config to point to the downloaded model
		modelPath := filepath.Join(*configPath, *modelName+".bin")
		transcriber.config.ModelPath = modelPath
		if err := transcriber.SaveConfig(); err != nil {
			fmt.Printf("Error saving updated configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Updated configuration to use model: %s\n", modelPath)

	case "stop":
		if err := killAllProcesses(); err != nil {
			fmt.Printf("Error stopping processes: %v\n", err)
			os.Exit(1)
		}
	}
}
