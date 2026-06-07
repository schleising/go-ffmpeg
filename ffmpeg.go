package go_ffmpeg

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

	// Context used to cancel the ffmpeg command
	ctx context.Context

	cleanupOnce sync.Once
}

type format struct {
	// The duration of the input file
	Duration string `json:"duration"`
}

type ffProbeOutput struct {
	// The format of the input file
	Format format `json:"format"`
}

func defaultOutputFile(inputFile string) string {
	outputFile := filepath.Join(filepath.Dir(inputFile), "Converted", filepath.Base(inputFile))
	return strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".mp4"
}

func NewFfmpeg(cancelContext context.Context, inputFile, outputFile string, command []string) (*Ffmpeg, error) {
	// Check if the input file exists
	if _, err := os.Stat(inputFile); err != nil {
		return nil, err
	}

	if outputFile == "" {
		outputFile = defaultOutputFile(inputFile)
	}

	// If the output file already exists, return an error
	if _, err := os.Stat(outputFile); err == nil {
		return nil, ErrOutputFileExists
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// Create the output directory if it does not exist
	outputDirectory := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDirectory, os.ModePerm); err != nil {
		return nil, err
	}

	// Build the command line options
	options := []string{
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
	defer ffprobeOutput.Close()

	// Start the ffprobe command
	if err = ffprobe.Start(); err != nil {
		return nil, ErrFfProbeCommand
	}

	// Create a scanner to read the output
	ffprobeOutputScanner := bufio.NewScanner(ffprobeOutput)

	// Read the output
	var outputBuilder strings.Builder
	for ffprobeOutputScanner.Scan() {
		outputBuilder.WriteString(strings.TrimSpace(ffprobeOutputScanner.Text()))
	}
	outputString := outputBuilder.String()
	if err = ffprobeOutputScanner.Err(); err != nil {
		return nil, ErrFfProbeRead
	}
	if err = ffprobe.Wait(); err != nil {
		return nil, ErrFfProbeRead
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
		ctx:        cancelContext,
	}

	// Return the ffmpeg struct
	return ffmpeg, nil
}

func (f *Ffmpeg) cleanUp() {
	f.cleanupOnce.Do(func() {
		// If the context was cancelled, delete the output file and send false to the done channel
		select {
		case <-f.ctx.Done():
			os.Remove(f.outputFile)

			select {
			case f.Done <- false:
			default:
			}
		default:
			select {
			case f.Done <- true:
				select {
				case f.Progress <- Progress{}:
				default:
				}
			default:
			}
		}

		close(f.Progress)
		close(f.Error)
		close(f.Done)
	})
}

func (f *Ffmpeg) monitorProgress(stderr io.ReadCloser) {
	defer stderr.Close()

	stdErrScanner := bufio.NewReader(stderr)

	for {
		line, err := stdErrScanner.ReadString('\r')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				select {
				case f.Error <- err:
				default:
				}
			}
			return
		}

		progress, err := newProgress(line, f.duration, f.startTime, f.inputFile, f.outputFile)
		if err != nil {
			if errors.Is(err, ErrNoProgressInformation) {
				continue
			}

			select {
			case f.Error <- err:
			default:
			}

			f.command.Cancel()
			return
		}

		select {
		case f.Progress <- *progress:
		default:
		}
	}
}

// Start launches the ffmpeg process and begins monitoring progress.
// Call Wait to block until the process exits and channels are closed.
func (f *Ffmpeg) Start() error {
	stderr, err := f.command.StderrPipe()
	if err != nil {
		return ErrStdErrPipe
	}

	go f.monitorProgress(stderr)

	return f.command.Start()
}

// Wait blocks until the ffmpeg process exits and performs cleanup.
func (f *Ffmpeg) Wait() error {
	err := f.command.Wait()
	f.cleanUp()
	return err
}

// Run starts ffmpeg and blocks until it completes.
func (f *Ffmpeg) Run() error {
	if err := f.Start(); err != nil {
		return err
	}
	return f.Wait()
}
