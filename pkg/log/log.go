// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package log

import (
	"os"

	"github.com/sirupsen/logrus"
)

// log object for logging across the pkg/
var log = logrus.New()

// SetupGlobal set ups a global logger
// TODO: Convert it to local instance shared by all BotKube components
// 	See: https://github.com/infracloudio/botkube/issues/589
func SetupGlobal() {
	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		// Set Info level as a default
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)
	log.Formatter = &logrus.TextFormatter{ForceColors: true, FullTimestamp: true}
}

// Info map logrus.Info func to log.Info
func Info(message ...interface{}) {
	log.Info(message...)
}

// Trace map logrus.Trace func to log.Trace
func Trace(message ...interface{}) {
	log.Trace(message...)
}

// Debug map logrus.Debug func to log.Debug
func Debug(message ...interface{}) {
	log.Debug(message...)
}

// Warn map logrus.Warn func to log.Warn
func Warn(message ...interface{}) {
	log.Warn(message...)
}

// Error map logrus.Error func to log.Error
func Error(message ...interface{}) {
	log.Error(message...)
}

// Fatal map logrus.Fatal func to log.Fatal
func Fatal(message ...interface{}) {
	log.Fatal(message...)
}

// Panic map logrus.Panic func to log.Panic
func Panic(message ...interface{}) {
	log.Panic(message...)
}

// Infof map logrus.Infof func to log.Infof
func Infof(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// Tracef map logrus.Tracef func to log.Tracef
func Tracef(format string, v ...interface{}) {
	log.Tracef(format, v...)
}

// Debugf map logrus.Debugf func to log.Debugf
func Debugf(format string, v ...interface{}) {
	log.Debugf(format, v...)
}

// Warnf map logrus.Warnf func to log.Warnf
func Warnf(format string, v ...interface{}) {
	log.Warnf(format, v...)
}

// Errorf map logrus.Errorf func to log.Errorf
func Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

// Fatalf map logrus.Fatalf func to log.Fatalf
func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// Panicf map logrus.Panicf func to log.Panicf
func Panicf(format string, v ...interface{}) {
	log.Panicf(format, v...)
}

//GetLevel logrus logLevel
func GetLevel() logrus.Level {
	return log.GetLevel()
}
