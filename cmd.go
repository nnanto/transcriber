package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"
)

func printUsage() {
	fmt.Printf("Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Println("Commands:")
	fmt.Println("  quick     Quick transcribe mode - record and transcribe immediately")
	fmt.Println("  process   Process existing MP3 files in a directory")
	fmt.Println("  config    Show current configuration and config file location")
	fmt.Println("  help      Show this help message")
	fmt.Println("\nOptions:")
	fmt.Println("  -output string")
	fmt.Println("        Output directory for transcriptions (default \".\")")
	fmt.Println("  -input string")
	fmt.Println("        Input directory for processing (defaults to temp directory)")
	fmt.Println("  -config string")
	fmt.Println("        Path to configuration file (defaults to standard config location)")
	fmt.Println("  -duration string")
	fmt.Println("        Recording duration for quick mode (e.g., 30s, 2m, 1h) (default \"30m\")")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s quick -output ./transcriptions\n", os.Args[0])
	fmt.Printf("  %s quick -duration 2m -output ./transcriptions\n", os.Args[0])
	fmt.Printf("  %s config\n", os.Args[0])
	fmt.Printf("  %s process -input /tmp/audio -output ./transcriptions\n", os.Args[0])
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

	// Validate command
	validCommands := map[string]bool{
		"quick":   true,
		"process": true,
		"config":  true,
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
		inputDir   = flagSet.String("input", "", "Input directory for processing (defaults to temp directory)")
		configPath = flagSet.String("config", "", "Path to configuration file (defaults to standard config location)")
		duration   = flagSet.Duration("duration", 30*time.Minute, "Recording duration for quick mode (e.g., 30s, 2m, 1h)")
	)

	flagSet.Usage = printUsage
	flagSet.Parse(os.Args[2:])

	transcriber, err := NewTranscriber(*configPath)
	if err != nil {
		fmt.Printf("Error initializing transcriber: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "quick":
		printProcessInfo()
		fmt.Printf("Recording duration: %v\n", *duration)
		if err := transcriber.QuickTranscribe(*outputDir, false, int(duration.Seconds())); err != nil {
			fmt.Printf("Error in quick transcribe: %v\n", err)
			os.Exit(1)
		}

	case "process":
		processInputDir := *inputDir
		if processInputDir == "" {
			processInputDir = transcriber.GetTempDir()
		}
		if err := transcriber.ProcessFiles(processInputDir, *outputDir); err != nil {
			fmt.Printf("Error processing files: %v\n", err)
			os.Exit(1)
		}

	case "config":
		config := transcriber.GetConfig()
		configJSON, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("Current configuration:\n%s\n\n", string(configJSON))
		fmt.Printf("Config file location: %s\n", transcriber.GetConfigPath())
		fmt.Println("To update configuration, edit the config file directly and restart the application.")
	}
}
