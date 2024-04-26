package go_ffmpeg

import (
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Progress struct to parse and store the progress of the ffmpeg command
type Progress struct {
	// The input file
	InputFile string `json:"input_file"`

	// The output file
	OutputFile string `json:"output_file"`

	// Frame number
	Frame int `json:"frame"`

	// Frames per second
	FPS float64 `json:"fps"`

	// Q value
	Q float64 `json:"q"`

	// Size of the output file
	Size float64 `json:"size"`

	// Time through the file
	Time time.Duration `json:"time"`

	// Bitrate
	Bitrate float64 `json:"bitrate"`

	// Duplicate frame count
	Dup int `json:"dup"`

	// Dropped frame count
	Drop int `json:"drop"`

	// Conversion speed
	Speed float64 `json:"speed"`

	// Percent complete
	PercentComplete float64 `json:"percent_complete"`

	// Time remaining
	TimeRemaining time.Duration `json:"time_remaining"`

	// Estimated finish time
	EstimatedFinishTime time.Time `json:"estimated_finish_time"`
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
		return nil, ErrNoProgressInformation
	}

	// Fields function to split the line
	fieldsFunc := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '-' && c != ':' && c != '/'
	}

	// Split the line
	fields := strings.FieldsFunc(line, fieldsFunc)

	// Check if the line contains the correct number of fields
	if len(fields) != 18 {
		return nil, ErrWrongNumberOfFields
	}

	// Parse the frame number
	frame, err := strconv.Atoi(fields[FrameIndex])
	if err != nil {
		return nil, ErrFrameNumber
	}

	// Parse the FPS
	fps, err := strconv.ParseFloat(fields[FPSIndex], 64)
	if err != nil {
		return nil, ErrFPS
	}

	// Parse the Q value
	q, err := strconv.ParseFloat(fields[QIndex], 64)
	if err != nil {
		return nil, ErrQ
	}

	// Parse the size
	size, err := strconv.ParseFloat(strings.TrimRight(fields[SizeIndex], "KiB"), 64)
	if err != nil {
		return nil, ErrSize
	}

	// Parse the time
	splitTime := strings.Split(fields[TimeIndex], ":")
	if len(splitTime) != 3 {
		return nil, ErrTime
	}

	// Get the hours, minutes, and seconds
	hours, err := strconv.Atoi(splitTime[0])
	if err != nil {
		return nil, ErrTime
	}

	minutes, err := strconv.Atoi(splitTime[1])
	if err != nil {
		return nil, ErrTime
	}

	seconds, err := strconv.ParseFloat(splitTime[2], 64)
	if err != nil {
		return nil, ErrTime
	}

	// Calculate the time through the file
	timeThroughFile := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second

	// Parse the bitrate
	bitrate, err := strconv.ParseFloat(strings.TrimRight(fields[BitrateIndex], "kbit/s"), 64)
	if err != nil {
		return nil, ErrBitrate
	}

	// Parse the dupliate frame count
	dup, err := strconv.Atoi(fields[DupIndex])
	if err != nil {
		return nil, ErrDup
	}

	// Parse the dropped frame count
	drop, err := strconv.Atoi(fields[DropIndex])
	if err != nil {
		return nil, ErrDrop
	}

	// Parse the speed
	speed, err := strconv.ParseFloat(strings.TrimRight(fields[SpeedIndex], "x"), 64)
	if err != nil {
		return nil, ErrSpeed
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
