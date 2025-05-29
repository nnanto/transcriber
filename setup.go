package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type progressWriter struct {
	file    *os.File
	total   int64
	written int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.file.Write(p)
	if err != nil {
		return n, err
	}

	pw.written += int64(n)
	if pw.total > 0 {
		percent := float64(pw.written) / float64(pw.total) * 100
		writtenMB := float64(pw.written) / (1024 * 1024)
		totalMB := float64(pw.total) / (1024 * 1024)
		fmt.Printf("\rProgress: %.1f%% (%.1fMB/%.1fMB)", percent, writtenMB, totalMB)
	} else {
		writtenMB := float64(pw.written) / (1024 * 1024)
		fmt.Printf("\rDownloaded: %.1fMB", writtenMB)
	}

	return n, err
}

func downloadModel(modelName, configPath string) error {
	// Ensure model name has .bin extension
	if !strings.HasSuffix(modelName, ".bin") {
		modelName += ".bin"
	}

	// Create models directory in config path
	modelsDir := configPath

	// Check if file already exists
	outputPath := filepath.Join(modelsDir, modelName)
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("Model already exists, skipping download: %s\n", outputPath)
		return nil
	}

	baseURL := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/"
	downloadURL := baseURL + modelName

	fmt.Printf("Downloading model from: %s\n", downloadURL)

	// Download the file
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download model: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download model: HTTP %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()

	// Copy with progress
	fmt.Printf("Saving to: %s\n", outputPath)

	pw := &progressWriter{
		file:  out,
		total: resp.ContentLength,
	}

	_, err = io.Copy(pw, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save model: %v", err)
	}

	fmt.Printf("\nModel downloaded successfully: %s\n", outputPath)
	return nil
}
