package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	ModelPath                  string `json:"model_path"`
	Language                   string `json:"language"`
	TempDir                    string `json:"temp_dir"`
	OutputFormat               string `json:"output_format"`
	WhisperCmd                 string `json:"whisper_cmd"`
	RecordingCmd               string `json:"recording_cmd"`
	ChunkDurationInSecs        int    `json:"chunk_duration_in_secs"`         // Duration in seconds for each chunk
	MinRequiredUniqueWordCount int    `json:"min_required_unique_word_count"` // Minimum unique words to process a chunk
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
		configPath = workDir
	}

	// Ensure config path is created
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath = filepath.Join(configPath, "config.json")
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
		ModelPath:                  filepath.Join(workDir, "ggml-large-v3-turbo-q5_0.bin"),
		Language:                   "English",
		TempDir:                    "/tmp/transcriber",
		OutputFormat:               "txt",
		WhisperCmd:                 "whisper-cli",
		RecordingCmd:               "ffmpeg",
		ChunkDurationInSecs:        30, // Default 30 seconds per chunk
		MinRequiredUniqueWordCount: 5,  // Minimum unique words to process a chunk
	}

	data, err := os.ReadFile(t.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return t.SaveConfig() // Create default config
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
	if loadedConfig.ChunkDurationInSecs > 0 {
		t.config.ChunkDurationInSecs = loadedConfig.ChunkDurationInSecs
	}

	return t.ensureTempDir()
}

func (t *Transcriber) SaveConfig() error {
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
	return t.recorder.Record(outputFile, duration)
}

func (t *Transcriber) transcribeAudioChunk(audioFile, outputPath string, chunkNum int, removeAudioFileOnSuccess bool) error {
	// Create temporary output file for this chunk
	tempOutputPath := outputPath + fmt.Sprintf("_chunk_%d", chunkNum)

	err := t.whisperService.Transcribe(audioFile, tempOutputPath)
	if err != nil {
		return fmt.Errorf("transcription failed for chunk %d: %v", chunkNum, err)
	}

	// Append chunk transcription to main output file
	chunkFile := tempOutputPath + "." + t.config.OutputFormat
	mainFile := outputPath + "." + t.config.OutputFormat

	if err := t.appendTranscription(chunkFile, mainFile, chunkNum); err != nil {
		return fmt.Errorf("failed to append chunk %d: %v", chunkNum, err)
	}

	// Clean up chunk files
	os.Remove(chunkFile)
	if removeAudioFileOnSuccess {
		os.Remove(audioFile)
	}

	return nil
}

func (t *Transcriber) appendTranscription(chunkFile, mainFile string, chunkNum int) error {
	// Read chunk transcription
	chunkData, err := os.ReadFile(chunkFile)
	if err != nil {
		return err
	}

	// If number of unique words in chunk is < 5, skip appending
	// Check for unique words
	if t.shouldSkipChunk(chunkData, chunkNum) {
		return nil
	}

	// Open main file for appending
	f, err := os.OpenFile(mainFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Calculate timestamp for this chunk
	startSeconds := (chunkNum - 1) * t.config.ChunkDurationInSecs
	endSeconds := chunkNum * t.config.ChunkDurationInSecs

	// Format timestamp as MM:SS or HH:MM:SS
	startTime := t.formatTimestamp(startSeconds)
	endTime := t.formatTimestamp(endSeconds)

	// Add chunk separator and content
	if chunkNum > 1 {
		f.WriteString("\n\n")
	}
	f.WriteString(fmt.Sprintf("[%s - %s]\n", startTime, endTime))
	f.Write(chunkData)

	return nil
}

func (t *Transcriber) shouldSkipChunk(chunkData []byte, chunkNum int) bool {
	words := strings.Fields(string(chunkData))
	uniqueWords := make(map[string]bool)
	for _, word := range words {
		uniqueWords[strings.ToLower(word)] = true
	}

	if len(uniqueWords) < t.config.MinRequiredUniqueWordCount {
		fmt.Printf("Skipping chunk %d due to insufficient unique words (%d found)\n", chunkNum, len(uniqueWords))
		return true
	}
	return false
}

func (t *Transcriber) formatTimestamp(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func (t *Transcriber) runTranscribe(outputDir string, removeAudioFileOnSuccess bool) error {
	sessionID := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, "run_"+sessionID)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	fmt.Printf("\n📝 Run this for Live transcription every %v secs: `tail -f %s.%s`\n\n",
		t.config.ChunkDurationInSecs, outputPath, t.config.OutputFormat)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Starting chunked transcription. Chunk size: %d seconds\n",
		t.config.ChunkDurationInSecs)
	fmt.Println("Press Ctrl+C to stop recording.")

	// Channel to communicate audio files for transcription
	audioFileChan := make(chan string, 2) // Buffer for 2 files
	transcriptionDone := make(chan struct{})

	// Start transcription goroutine
	go func() {
		defer close(transcriptionDone)
		chunkNum := 1
		for audioFile := range audioFileChan {
			// Check if we have a valid recording
			if info, err := os.Stat(audioFile); err != nil || info.Size() == 0 {
				fmt.Printf("Warning: No valid recording for chunk %d, skipping\n", chunkNum)
				chunkNum++
				continue
			}

			// Transcribe this chunk and append to main file
			if err := t.transcribeAudioChunk(audioFile, outputPath, chunkNum, removeAudioFileOnSuccess); err != nil {
				fmt.Printf("Error processing chunk %d: %v\n", chunkNum, err)
				continue
			}
			chunkNum++
		}
	}()

	chunkNum := 1

	for {
		// Check for interrupt signal before starting new chunk
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal. Stopping transcription...")
			close(audioFileChan) // Stop sending new files for transcription
			<-transcriptionDone  // Wait for transcription to finish
			if chunkNum > 1 {
				fmt.Printf("Transcription saved to: %s.%s\n", outputPath, t.config.OutputFormat)
			}
			return nil
		default:
			// Continue with recording
		}

		chunkDuration := t.config.ChunkDurationInSecs

		audioFile := filepath.Join(t.config.TempDir, fmt.Sprintf("chunk_%s_%d.mp3", sessionID, chunkNum))

		fmt.Printf("Recording chunk %d (every %d seconds)...\n", chunkNum, chunkDuration)

		// Record this chunk
		if err := t.recordAudio(audioFile, chunkDuration); err != nil {
			// Check if error was due to interrupt
			select {
			case <-sigChan:
				fmt.Println("\nRecording interrupted. Stopping transcription...")
				close(audioFileChan)
				<-transcriptionDone
				if chunkNum > 1 {
					fmt.Printf("Transcription saved to: %s.%s\n", outputPath, t.config.OutputFormat)
				}
				return nil
			default:
				return fmt.Errorf("recording error for chunk %d: %v", chunkNum, err)
			}
		}

		// Send audio file for transcription (non-blocking)
		select {
		case audioFileChan <- audioFile:
			// File sent successfully
		default:
			// Channel full, wait a bit and try again
			fmt.Printf("Transcription queue full, waiting...\n")
			audioFileChan <- audioFile
		}

		chunkNum++
	}

}

// Export methods for use in cmd.go
func (t *Transcriber) RunTranscribe(outputDir string, removeAudioFileOnSuccess bool) error {
	return t.runTranscribe(outputDir, removeAudioFileOnSuccess)
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
