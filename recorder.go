package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const MAX_RECORD_DURATION_IN_SECS = 30 * 60

type Recorder struct {
	stopChan      chan struct{}
	device        string
	displayOutput bool
}

func NewRecorder(device string, displayOutput bool) *Recorder {
	return &Recorder{
		stopChan:      make(chan struct{}),
		device:        device,
		displayOutput: displayOutput,
	}
}

func NewRecorderWithDefaultDevice(displayOutput bool) *Recorder {
	device := getDefaultDevice()
	return &Recorder{
		stopChan:      make(chan struct{}),
		device:        device,
		displayOutput: displayOutput,
	}
}

func (r *Recorder) getFFmpegCommand(outputFile string, duration int) *exec.Cmd {
	if duration <= 0 {
		duration = MAX_RECORD_DURATION_IN_SECS // Default to 10 seconds if no duration is specified
	}
	switch runtime.GOOS {
	case "darwin": // macOS
		return exec.Command("ffmpeg",
			"-f", "avfoundation",
			"-i", r.device,
			"-t", fmt.Sprintf("%d", duration),
			"-y",
			outputFile,
		)
	case "linux":
		return exec.Command("ffmpeg",
			"-f", "alsa",
			"-i", r.device,
			"-t", fmt.Sprintf("%d", duration),
			"-y",
			outputFile,
		)
	case "windows":
		return exec.Command("ffmpeg",
			"-f", "dshow",
			"-i", fmt.Sprintf("audio=%s", r.device),
			"-t", fmt.Sprintf("%d", duration),
			"-y",
			outputFile,
		)
	default:
		// Fallback - try pulse for other Unix-like systems
		return exec.Command("ffmpeg",
			"-f", "pulse",
			"-i", r.device,
			"-t", fmt.Sprintf("%d", duration),
			"-y",
			outputFile,
		)
	}
}

func (r *Recorder) isCleanExit(err error) bool {
	if err == nil {
		return true
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		status := exitErr.ExitCode()
		// Common clean exit codes for FFmpeg
		switch status {
		case 0: // Normal exit
			return true
		case 255: // FFmpeg quit command
			return true
		case 130: // SIGINT (Ctrl+C)
			return true
		case 143: // SIGTERM
			return true
		}
	}
	return false
}

func (r *Recorder) gracefulStop(cmd *exec.Cmd, stdin io.WriteCloser, done chan error) error {
	fmt.Println("Stopping recording gracefully...")

	// Send quit command to FFmpeg
	if _, err := stdin.Write([]byte("q")); err != nil {
		fmt.Printf("Warning: Could not send quit command: %v\n", err)
	}
	stdin.Close()

	// Wait for FFmpeg to exit gracefully
	time.Sleep(500 * time.Millisecond)

	select {
	case err := <-done:
		if r.isCleanExit(err) {
			return nil
		}
		return fmt.Errorf("FFmpeg error: %v", err)
	case <-time.After(3 * time.Second):
		fmt.Println("FFmpeg taking too long to exit, terminating...")
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("ffmpeg failed to exit cleanly")
	}
}

func (r *Recorder) Record(outputFile string, duration int) error {
	cmd := r.getFFmpegCommand(outputFile, duration)

	// Set up stdout/stderr
	if r.displayOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %v", err)
	}

	// Monitor process completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v\n", sig)
		return r.gracefulStop(cmd, stdin, done)
	case err := <-done:
		if r.isCleanExit(err) {
			return nil
		}
		return fmt.Errorf("recording failed: %v", err)
	}
}

func (r *Recorder) Stop() {
	// Implementation depends on your recorder - this should interrupt the ongoing recording
	// For example, if using a process, send a signal to stop it
	// This is a placeholder - implement based on your actual recorder implementation
}

func getDefaultDevice() string {
	switch runtime.GOOS {
	case "darwin":
		return ":MacBook Pro Microphone"
	case "linux":
		return "default"
	case "windows":
		return "Microphone"
	default:
		return "default"
	}
}
