package go_ffmpeg

import "errors"

// Errors that can be returned by the progress parser
var (
	ErrNoProgressInformation = errors.New("no progress information")
	ErrWrongNumberOfFields   = errors.New("line does not contain the correct number of fields")
	ErrFrameNumber           = errors.New("could not parse frame number")
	ErrFPS                   = errors.New("could not parse FPS")
	ErrQ                     = errors.New("could not parse Q value")
	ErrSize                  = errors.New("could not parse size")
	ErrTime                  = errors.New("could not parse time")
	ErrBitrate               = errors.New("could not parse bitrate")
	ErrDup                   = errors.New("could not parse dup")
	ErrDrop                  = errors.New("could not parse drop")
	ErrSpeed                 = errors.New("could not parse speed")
)

// Errors that can be returned by the ffmpeg command
var (
	ErrFfProbeStdOutPipe = errors.New("could not create ffprobe stdout pipe")
	ErrStdErrPipe        = errors.New("could not create stderr pipe")
	ErrFfProbeCommand    = errors.New("could not create ffprobe command")
	ErrFfProbeDuration   = errors.New("could not get duration from ffprobe")
)
