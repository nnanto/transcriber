package main

import (
	"fmt"
	"os"
	"os/exec"
)

type WhisperService struct {
	config *Config
}

func NewWhisperService(config *Config) *WhisperService {
	return &WhisperService{
		config: config,
	}
}

func (w *WhisperService) ValidateModel() error {
	if _, err := os.Stat(w.config.ModelPath); err != nil {
		return fmt.Errorf("model file not found: %s", w.config.ModelPath)
	}
	return nil
}

func (w *WhisperService) Transcribe(audioFile, outputPath string) error {
	if _, err := os.Stat(audioFile); err != nil {
		return fmt.Errorf("audio file not accessible: %v", err)
	}

	if err := w.ValidateModel(); err != nil {
		return err
	}

	outputFlag := "--output-" + w.config.OutputFormat
	cmd := exec.Command(w.config.WhisperCmd,
		audioFile,
		"-m", w.config.ModelPath,
		"--language", w.config.Language,
		outputFlag,
		"-of", outputPath,
	)

	fmt.Printf("Transcribing: %s\n", audioFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("transcription failed: %v", err)
	}

	expectedFile := outputPath + "." + w.config.OutputFormat
	if _, err := os.Stat(expectedFile); err != nil {
		return fmt.Errorf("transcription output not found: %s", expectedFile)
	}

	fmt.Printf("Transcription saved: %s\n", expectedFile)
	return nil
}
