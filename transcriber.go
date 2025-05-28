package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	ModelPath    string `json:"model_path"`
	Language     string `json:"language"`
	TempDir      string `json:"temp_dir"`
	OutputFormat string `json:"output_format"`
	WhisperCmd   string `json:"whisper_cmd"`
	RecordingCmd string `json:"recording_cmd"`
}

type Transcriber struct {
	config         Config
	configPath     string
	stopChan       chan struct{}
	recorder       *Recorder
	whisperService *WhisperService
}

func NewTranscriber(configPath string) (*Transcriber, error) {
	if configPath == "" {
		workDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %v", err)
		}
		configPath = filepath.Join(workDir, "config.json")
	}

	t := &Transcriber{
		configPath: configPath,
		stopChan:   make(chan struct{}),
	}

	if err := t.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	// Initialize recorder with the configured audio input
	t.recorder = NewRecorderWithDefaultDevice(false)

	// Initialize whisper service
	t.whisperService = NewWhisperService(&t.config)

	return t, nil
}

func (t *Transcriber) loadConfig() error {
	workDir := filepath.Dir(t.configPath)
	// Set sensible defaults
	t.config = Config{
		ModelPath:    filepath.Join(workDir, "ggml-large-v3-turbo-q5_0.bin"),
		Language:     "English",
		TempDir:      "/tmp/transcriber",
		OutputFormat: "txt",
		WhisperCmd:   "whisper-cli",
		RecordingCmd: "ffmpeg",
	}

	data, err := os.ReadFile(t.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return t.saveConfig() // Create default config
		}
		return err
	}

	var loadedConfig Config
	if err := json.Unmarshal(data, &loadedConfig); err != nil {
		return err
	}

	// Merge loaded config with defaults
	if loadedConfig.ModelPath != "" {
		t.config.ModelPath = loadedConfig.ModelPath
	}
	if loadedConfig.Language != "" {
		t.config.Language = loadedConfig.Language
	}
	if loadedConfig.TempDir != "" {
		t.config.TempDir = loadedConfig.TempDir
	}
	if loadedConfig.OutputFormat != "" {
		t.config.OutputFormat = loadedConfig.OutputFormat
	}
	if loadedConfig.WhisperCmd != "" {
		t.config.WhisperCmd = loadedConfig.WhisperCmd
	}
	if loadedConfig.RecordingCmd != "" {
		t.config.RecordingCmd = loadedConfig.RecordingCmd
	}

	return t.ensureTempDir()
}

func (t *Transcriber) saveConfig() error {
	data, err := json.MarshalIndent(t.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.configPath, data, 0644)
}

func (t *Transcriber) ensureTempDir() error {
	return os.MkdirAll(t.config.TempDir, 0755)
}

// Remove the recordAudio method and replace with simpler recording
func (t *Transcriber) recordAudio(outputFile string, duration int) error {
	fmt.Printf("Recording audio to %s...\n", outputFile)
	return t.recorder.Record(outputFile, duration)
}

func (t *Transcriber) transcribeAudio(audioFile, outputPath string, removeAudioFileOnSuccess bool) error {
	err := t.whisperService.Transcribe(audioFile, outputPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %v", err)
	}
	if removeAudioFileOnSuccess {
		err := os.Remove(audioFile)
		if err != nil {
			return fmt.Errorf("failed to remove audio file: %v", err)
		}
		fmt.Printf("Removed audio file: %s\n", audioFile)
	}
	return nil
}

func (t *Transcriber) processAudioBlock(audioFile string, outputPath string, removeAudioFileOnSuccess bool) error {
	// Check if we have a valid recording
	if info, err := os.Stat(audioFile); err != nil || info.Size() == 0 {
		return fmt.Errorf("no valid recording found")
	}

	return t.transcribeAudio(audioFile, outputPath, removeAudioFileOnSuccess)
}

func (t *Transcriber) quickTranscribe(outputDir string, removeAudioFileOnSuccess bool, duration int) error {
	sessionID := time.Now().Format("20060102_150405")
	audioFile := filepath.Join(t.config.TempDir, "quick_"+sessionID+".mp3")
	outputPath := filepath.Join(outputDir, "quick_"+sessionID)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	fmt.Println("Starting quick transcribe mode. Press Ctrl+C to stop recording.")

	// Start recording and wait for completion
	if err := t.recordAudio(audioFile, duration); err != nil {
		return fmt.Errorf("recording error: %v", err)
	}

	// Process the recording
	return t.processAudioBlock(audioFile, outputPath, removeAudioFileOnSuccess)
}

// Export methods for use in cmd.go
func (t *Transcriber) QuickTranscribe(outputDir string, removeAudioFileOnSuccess bool, duration int) error {
	return t.quickTranscribe(outputDir, removeAudioFileOnSuccess, duration)
}

func (t *Transcriber) processFiles(inputDir, outputDir string) error {
	pattern := filepath.Join(inputDir, "*.mp3")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Printf("No MP3 files found in %s\n", inputDir)
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	for _, file := range files {
		basename := filepath.Base(file)
		name := basename[:len(basename)-4] // Remove .mp3 extension
		outputPath := filepath.Join(outputDir, name)

		fmt.Printf("Processing: %s\n", file)
		if err := t.transcribeAudio(file, outputPath, true); err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
			continue
		}

		if err := os.Remove(file); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", file, err)
		}
	}

	fmt.Println("Processing completed.")
	return nil
}

func (t *Transcriber) ProcessFiles(inputDir, outputDir string) error {
	return t.processFiles(inputDir, outputDir)
}

func (t *Transcriber) GetConfig() Config {
	return t.config
}

func (t *Transcriber) GetConfigPath() string {
	return t.configPath
}

func (t *Transcriber) GetTempDir() string {
	return t.config.TempDir
}
