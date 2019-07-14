package Leaflog

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"
	"errors"

)

//levels
const (
	debugLevel   = 0
	releaseLevel = 1
	errorLevel   = 2
	fatalLevel   = 3
)

const (
	printDebugLevel   = "[debug  ]"
	printReleaseLevel = "[release]"
	printErrorLevel   = "[error  ]"
	printFatalLevel   = "[fatal  ]"
)

type Logger struct {
	level      int //debug等级
	baseLogger *log.Logger
	baseFile   *os.File
}

func New(strLevel, pathname string, flag int) (*Logger, error) {
	//level
	var level int
	switch strings.ToLower(strLevel) {
	case "debug":
		level = debugLevel
	case "release":
		level = releaseLevel
	case "error":
		level = errorLevel
	case "fatal":
		level = fatalLevel
	default:
		return nil, errors.New("unknown level:" + strLevel)
	}

	//logger
	var baseLogger *log.Logger
	var baseFile *os.File

	if pathname != "" {
		now := time.Now()

		filename := fmt.Sprintln("%d%02d%02d_%02d_%02d_%02d.log",
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			now.Second())
		file, err := os.Create(path.Join(pathname, filename))
		if err != nil {
			return nil, err
		}
		baseLogger = log.New(file, "", flag)
		baseFile = file
	} else {
		baseLogger = log.New(os.Stdout, "", flag)
	}

	//new
	logger := new(Logger)
	logger.level = level
	logger.baseLogger = baseLogger
	logger.baseFile = baseFile

	return logger, nil
}

// It's dangerous to call the method on logging
func (logger *Logger) Close() {
	if logger.baseFile != nil {
		logger.baseFile.Close()
	}
}

func (logger *Logger) doPrintf(level int, printLevel string, format string, a ...interface{}) {
	if level < logger.level {
		return
	}

	if logger.baseLogger == nil {
		panic("logger closed")
	}

	format = printLevel + format

	logger.baseLogger.Output(3, fmt.Sprintf(format, a...))

	if level == fatalLevel {
		os.Exit(1)
	}
}

func (logger *Logger) Debug(format string, a ...interface{}) {
	logger.doPrintf(debugLevel, printDebugLevel, format, a...)
}

func (logger *Logger) Release(format string, a ...interface{}) {
	logger.doPrintf(releaseLevel, printReleaseLevel, format, a...)
}

func (logger *Logger) Error(format string, a ...interface{}) {
	logger.doPrintf(errorLevel, printErrorLevel, format, a...)
}

func (logger *Logger) Fatal(format string, a ...interface{}) {
	logger.doPrintf(fatalLevel, printFatalLevel, format, a...)
}

var gLogger, _ = New("debug", "", log.LstdFlags)

// It's dangerous to call the method on logging
func Export(logger *Logger) {
	if logger != nil {
		gLogger = logger
	}
}

func Dubug(format string, a ...interface{}) {
	gLogger.Debug(format, a)
}

func Release(format string, a ...interface{}) {
	gLogger.Release(format, a)
}

func Error(format string, a ...interface{}) {
	gLogger.Error(format, a)
}

func Fatal(format string, a ...interface{}) {
	gLogger.Fatal(format, a)
}

// It's dangerous to call the method on logging
func Close() {
	gLogger.Close()
}
