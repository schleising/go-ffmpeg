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
	InputFile string `json:"inputFile"`

	// The output file
	OutputFile string `json:"outputFile"`

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

	// Conversion speed
	Speed float64 `json:"speed"`

	// Percent complete
	PercentComplete float64 `json:"percentComplete"`

	// Time remaining
	TimeRemaining time.Duration `json:"timeRemaining"`

	// Estimated finish time
	EstimatedFinishTime time.Time `json:"estimatedFinishTime"`
}

// Parse the progress information from the ffmpeg stderr output
func newProgress(line string, duration time.Duration, startTime time.Time, inputFile string, outputFile string) (*Progress, error) {
	// Declare the indexes for the fields
	var frameIndex, fpsIndex, qIndex, sizeIndex, timeIndex, bitrateIndex, speedIndex int

	// Declare the fields
	var frame int
	var fps, q, size, bitrate, speed float64
	var timeThroughFile time.Duration
	var hours, minutes int
	var seconds float64

	// Declare the error
	var err error

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

	// Loop through the fields extracting the values
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)

		switch fields[i] {
		case "frame":
			frameIndex = i + 1
		case "fps":
			fpsIndex = i + 1
		case "q":
			qIndex = i + 1
		case "size":
			sizeIndex = i + 1
		case "time":
			timeIndex = i + 1
		case "bitrate":
			bitrateIndex = i + 1
		case "speed":
			speedIndex = i + 1
		}
	}

	// Parse the frame number
	if frameIndex != 0 && frameIndex < len(fields) {
		if frame, err = strconv.Atoi(fields[frameIndex]); err != nil {
			return nil, ErrFrameNumber
		}
	} else {
		return nil, ErrFrameNumber
	}

	// Parse the FPS
	if fpsIndex != 0 && fpsIndex < len(fields) {
		if fps, err = strconv.ParseFloat(fields[fpsIndex], 64); err != nil {
			return nil, ErrFPS
		}
	} else {
		return nil, ErrFPS
	}

	// Parse the Q value
	if qIndex != 0 && qIndex < len(fields) {
		if q, err = strconv.ParseFloat(fields[qIndex], 64); err != nil {
			return nil, ErrQ
		}
	} else {
		return nil, ErrQ
	}

	// Parse the size
	if sizeIndex != 0 && sizeIndex < len(fields) {
		if size, err = strconv.ParseFloat(strings.TrimRight(fields[sizeIndex], "KiB"), 64); err != nil {
			return nil, ErrSize
		}
	} else {
		return nil, ErrSize
	}

	// Parse the time
	if timeIndex != 0 && timeIndex < len(fields) {
		// Set the time to 0 if it is N/A
		if fields[timeIndex] == "N/A" {
			timeThroughFile = time.Duration(0)
		} else {
			// Split the time into hours, minutes, and seconds
			splitTime := strings.Split(fields[timeIndex], ":")

			if len(splitTime) != 3 {
				return nil, ErrTime
			}
			// Get the hours, minutes, and seconds
			if hours, err = strconv.Atoi(splitTime[0]); err != nil {
				return nil, ErrTime
			}

			if minutes, err = strconv.Atoi(splitTime[1]); err != nil {
				return nil, ErrTime
			}

			if seconds, err = strconv.ParseFloat(splitTime[2], 64); err != nil {
				return nil, ErrTime
			}

			// Calculate the time through the file
			timeThroughFile = time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
		}
	} else {
		return nil, ErrTime
	}

	// Parse the bitrate
	if bitrateIndex != 0 && bitrateIndex < len(fields) {
		// Handle the case where the bitrate is N/A
		if fields[bitrateIndex] == "N/A" {
			bitrate = 0
		} else {
			if bitrate, err = strconv.ParseFloat(strings.TrimRight(fields[bitrateIndex], "kbit/s"), 64); err != nil {
				return nil, ErrBitrate
			}
		}
	} else {
		return nil, ErrBitrate
	}

	// Parse the speed
	if speedIndex != 0 && speedIndex < len(fields) {
		// Handle the case where the speed is N/A
		if fields[speedIndex] == "N/A" {
			speed = 0
		} else {
			if speed, err = strconv.ParseFloat(strings.TrimRight(fields[speedIndex], "x"), 64); err != nil {
				return nil, ErrSpeed
			}
		}
	} else {
		return nil, ErrSpeed
	}

	// Calculate the percent complete
	percentComplete := float64(timeThroughFile) / float64(duration) * 100

	// Calculate the time taken
	timeTaken := time.Since(startTime)

	// Prevent division by zero
	if percentComplete == 0 {
		percentComplete = 0.01
	}

	// Calculate the time remaining
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
