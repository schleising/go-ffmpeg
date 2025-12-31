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

type format struct {
	// The duration of the input file
	Duration string `json:"duration"`
}

type ffProbeOutput struct {
	// The format of the input file
	Format format `json:"format"`
}

func NewFfmpeg(cancelContext context.Context, inputFile string, command []string) (*Ffmpeg, error) {
	// Check if the input file exists
	_, err := os.Stat(inputFile)
	if os.IsNotExist(err) {
		return nil, err
	}

	// Set the output file to the Converted subdirectory of the directory the input file is in with the same name as the input file
	outputFile := filepath.Join(filepath.Dir(inputFile), "Converted", filepath.Base(inputFile))

	// Change the output file extension to .mp4
	outputFile = strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".mp4"

	// If the output file already exists, return an error
	_, err = os.Stat(outputFile)
	if !os.IsNotExist(err) {
		return nil, ErrOutputFileExists
	}

	// Create the output directory if it does not exist
	outputDirectory := filepath.Dir(outputFile)
	if err = os.MkdirAll(outputDirectory, os.ModePerm); err != nil {
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

	// Get the input file details with ffprobe
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
	if err = ffprobe.Start(); err != nil {
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
	var output ffProbeOutput
	if err = json.Unmarshal([]byte(outputString), &output); err != nil {
		return nil, ErrFfProbeUnmarshal
	}

	// Convert the duration string to a float64
	durationSeconds, err := strconv.ParseFloat(output.Format.Duration, 64)
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
	// If the context was cancelled, delete the output file and send true or false to the done channel
	select {
	case <-f.context.Done():
		os.Remove(f.outputFile)

		// Signal that the ffmpeg command is done
		select {
		case f.Done <- false:
		default:
		}
	default:
		// Signal that the ffmpeg command is done
		select {
		case f.Done <- true:
			// Send an empty progress struct to the channel
			select {
			case f.Progress <- Progress{}:
			default:
			}
		default:
		}
	}

	// Close the progress channel
	close(f.Progress)

	// Close the error channel
	close(f.Error)

	// Close the done channel
	close(f.Done)
}

func (f *Ffmpeg) Start() error {
	// Create a reader to read the output from stderr
	stderr, err := f.command.StderrPipe()
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
					select {
					case f.Error <- err:
					default:
					}

					// Cancel the ffmpeg command
					f.command.Cancel()

					// Clean up
					f.cleanUp()

					// Return
					return
				} else {
					// Continue to the next iteration
					continue
				}
			}

			// Try to send the progress to the channel, if there is no listener continue to the next iteration
			select {
			case f.Progress <- *progress:
			default:
			}
		}
	}()

	// Run the command
	if err = f.command.Run(); err != nil {
		return err
	}

	return nil
}
