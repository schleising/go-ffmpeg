package go_ffmpeg

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Ffmpeg struct {
	// The input file
	inputFile string

	// The output file
	outputFile string

	// Ffmpeg command to run
	command *exec.Cmd

	// Duration of the input file
	duration time.Duration

	// Start time of the ffmpeg command
	startTime time.Time

	// Progress channel
	Progress chan Progress

	// Error channel
	Error chan error

	// Done channel
	Done chan bool

	// Cancel Context
	context context.Context
}

func NewFfmpeg(cancelContext context.Context, inputFile string, outputFile string, command []string) (*Ffmpeg, error) {
	// Check if the input file exists
	_, err := os.Stat(inputFile)
	if os.IsNotExist(err) {
		return nil, err
	}

	// Create the output directory if it does not exist
	outputDirectory := filepath.Dir(outputFile)
	err = os.MkdirAll(outputDirectory, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// Build the command line options
	options := []string{
		"-y",
		"-i",
		inputFile,
	}

	// Append the command options
	options = append(options, command...)

	// Append the output file
	options = append(options, outputFile)

	// Create a subprocess to run ffmpeg
	cmd := exec.CommandContext(cancelContext, "ffmpeg", options...)

	// Create a channel to send the progress
	progressChannel := make(chan Progress)

	// Create a channel to send errors
	errorChannel := make(chan error)

	// Create a channel to send done signal
	doneChannel := make(chan bool)

	// Get the input fiel details with ffprobe
	ffprobe := exec.Command(
		"ffprobe",
		"-print_format",
		"json",
		"-show_format",
		inputFile,
	)

	// Get the output pipe
	ffprobeOutput, err := ffprobe.StdoutPipe()
	if err != nil {
		return nil, ErrFfProbeStdOutPipe
	}

	// Defer closing the output pipe
	defer ffprobeOutput.Close()

	// Start the ffprobe command
	err = ffprobe.Start()
	if err != nil {
		return nil, ErrFfProbeCommand
	}

	// Create a scanner to read the output
	ffprobeOutputScanner := bufio.NewScanner(ffprobeOutput)

	// Read the output
	outputString := ""
	for ffprobeOutputScanner.Scan() {
		outputString += strings.TrimSpace(ffprobeOutputScanner.Text())
	}

	// Unmarshal the output
	var ffprobeOutputMap map[string]interface{}
	err = json.Unmarshal([]byte(outputString), &ffprobeOutputMap)
	if err != nil {
		return nil, err
	}

	// Get the duration of the input file
	durationString := ffprobeOutputMap["format"].(map[string]interface{})["duration"].(string)

	// Convert the duration string to a float64
	durationSeconds, err := strconv.ParseFloat(durationString, 64)
	if err != nil {
		return nil, ErrFfProbeDuration
	}

	// Convert the duration to a time.Duration
	duration := time.Duration(durationSeconds * float64(time.Second))

	// Create the ffmpeg struct
	ffmpeg := &Ffmpeg{
		inputFile:  inputFile,
		outputFile: outputFile,
		command:    cmd,
		duration:   duration,
		startTime:  time.Now(),
		Progress:   progressChannel,
		Error:      errorChannel,
		Done:       doneChannel,
		context:    cancelContext,
	}

	// Return the ffmpeg struct
	return ffmpeg, nil
}

func (f *Ffmpeg) cleanUp() {
	// Close the progress channel
	close(f.Progress)

	// If the context was cancelled, delete the output file
	select {
	case <-f.context.Done():
		os.Remove(f.outputFile)
	default:
	}

	// Close the error channel
	close(f.Error)

	// Signal that the ffmpeg command is done
	f.Done <- false

	// Close the done channel
	close(f.Done)
}

func (f *Ffmpeg) Start() error {
	// Create a reader to read the output from stderr
	stderr, err := f.command.StderrPipe()

	// Check for errors
	if err != nil {
		return ErrStdErrPipe
	}

	// Defer closing the stderr pipe
	defer stderr.Close()

	// Create a reader to read the output
	stdErrScanner := bufio.NewReader(stderr)

	// Start a goroutine to read the output
	go func() {
		// Read the output
		for {
			// Read the line
			line, err := stdErrScanner.ReadString('\r')
			if err != nil {
				// Cancel the ffmpeg command
				f.cleanUp()

				// Return
				return
			}

			// Log the output
			progress, err := newProgress(line, f.duration, f.startTime, f.inputFile, f.outputFile)
			if err != nil {
				// Do not send an error if the progress information is not available
				if !errors.Is(err, ErrNoProgressInformation) {
					// Send an error to the error channel
					f.Error <- err
				}

				// Continue to the next iteration
				continue
			}

			// Send the progress to the channel
			f.Progress <- *progress
		}
	}()

	// Run the command
	err = f.command.Run()
	if err != nil {
		return err
	}

	return nil
}
