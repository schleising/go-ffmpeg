package go_ffmpeg

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Progress struct to parse and store the progress of the ffmpeg command
type Progress struct {
	// The input file
	InputFile string

	// The output file
	OutputFile string

	// Frame number
	Frame int

	// Frames per second
	FPS float64

	// Q value
	Q float64

	// Size of the output file
	Size float64

	// Time through the file
	Time time.Duration

	// Bitrate
	Bitrate float64

	// Duplicate frame count
	Dup int

	// Dropped frame count
	Drop int

	// Conversion speed
	Speed float64

	// Percent complete
	PercentComplete float64

	// Time remaining
	TimeRemaining time.Duration

	// Estimated finish time
	EstimatedFinishTime time.Time
}

// Indices of the progress fields
const (
	FrameIndex   = 1
	FPSIndex     = 3
	QIndex       = 5
	SizeIndex    = 7
	TimeIndex    = 9
	BitrateIndex = 11
	DupIndex     = 13
	DropIndex    = 15
	SpeedIndex   = 17
)

// Parse the progress information from the ffmpeg stderr output
func newProgress(line string, duration time.Duration, startTime time.Time, inputFile string, outputFile string) (*Progress, error) {
	// Check if the line contains progress information
	if !strings.HasPrefix(line, "frame=") {
		return nil, errors.New("line does not contain progress information")
	}

	// Fields function to split the line
	fieldsFunc := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '-' && c != ':' && c != '/'
	}

	// Split the line
	fields := strings.FieldsFunc(line, fieldsFunc)

	// Check if the line contains the correct number of fields
	if len(fields) != 18 {
		return nil, errors.New("line does not contain the correct number of fields")
	}

	// Parse the frame number
	frame, err := strconv.Atoi(fields[FrameIndex])
	if err != nil {
		return nil, err
	}

	// Parse the FPS
	fps, err := strconv.ParseFloat(fields[FPSIndex], 64)
	if err != nil {
		return nil, err
	}

	// Parse the Q value
	q, err := strconv.ParseFloat(fields[QIndex], 64)
	if err != nil {
		return nil, err
	}

	// Parse the size
	size, err := strconv.ParseFloat(strings.TrimRight(fields[SizeIndex], "KiB"), 64)
	if err != nil {
		return nil, err
	}

	// Parse the time
	splitTime := strings.Split(fields[TimeIndex], ":")
	if len(splitTime) != 3 {
		return nil, errors.New("time does not contain hours, minutes, and seconds")
	}

	// Get the hours, minutes, and seconds
	hours, err := strconv.Atoi(splitTime[0])
	if err != nil {
		return nil, err
	}

	minutes, err := strconv.Atoi(splitTime[1])
	if err != nil {
		return nil, err
	}

	seconds, err := strconv.ParseFloat(splitTime[2], 64)
	if err != nil {
		return nil, err
	}

	// Calculate the time through the file
	timeThroughFile := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second

	// Parse the bitrate
	bitrate, err := strconv.ParseFloat(strings.TrimRight(fields[BitrateIndex], "kbit/s"), 64)
	if err != nil {
		return nil, err
	}

	// Parse the dupliate frame count
	dup, err := strconv.Atoi(fields[DupIndex])
	if err != nil {
		return nil, err
	}

	// Parse the dropped frame count
	drop, err := strconv.Atoi(fields[DropIndex])
	if err != nil {
		return nil, err
	}

	// Parse the speed
	speed, err := strconv.ParseFloat(strings.TrimRight(fields[SpeedIndex], "x"), 64)
	if err != nil {
		return nil, err
	}

	// Calculate the percent complete
	percentComplete := float64(timeThroughFile) / float64(duration) * 100

	// Calculate the time taken and time remaining
	timeTaken := time.Since(startTime)
	timeRemaining := time.Duration(float64(timeTaken) / percentComplete * (100 - percentComplete))

	// Calculate the estimated finish time
	predictedFinishTime := startTime.Add(timeTaken + timeRemaining)

	// Return the progress struct
	return &Progress{
		InputFile:           inputFile,
		OutputFile:          outputFile,
		Frame:               frame,
		FPS:                 fps,
		Q:                   q,
		Size:                size,
		Time:                timeThroughFile,
		Bitrate:             bitrate,
		Dup:                 dup,
		Drop:                drop,
		Speed:               speed,
		PercentComplete:     percentComplete,
		TimeRemaining:       timeRemaining,
		EstimatedFinishTime: predictedFinishTime,
	}, nil
}

// String method for the Progress struct
func (p Progress) String() string {
	return strconv.FormatFloat(p.PercentComplete, 'f', 2, 64) + "% Complete - " + "Time Remaining: " + p.TimeRemaining.Truncate(time.Second).String() + " - Estimated Finish Time: " + p.EstimatedFinishTime.Format(time.TimeOnly)
}
