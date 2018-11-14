/*
 * Minio Cloud Storage, (C) 2015, 2016, 2017, 2018 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logger

import (
	"context"
	"fmt"
	"go/build"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Disable disables all logging, false by default. (used for "go test")
var Disable = false

// Level type
type Level int8

// Enumerated level types
const (
	InformationLvl Level = iota + 1
	ErrorLvl
	FatalLvl
)

var trimStrings []string

// TimeFormat - logging time format.
const TimeFormat string = "15:04:05 MST 01/02/2006"

// List of error strings to be ignored by LogIf
const (
	diskNotFoundError = "disk not found"
)

var matchingFuncNames = [...]string{
	"http.HandlerFunc.ServeHTTP",
	"cmd.serverMain",
	"cmd.StartGateway",
	"cmd.(*webAPIHandlers).ListBuckets",
	"cmd.(*webAPIHandlers).MakeBucket",
	"cmd.(*webAPIHandlers).DeleteBucket",
	"cmd.(*webAPIHandlers).ListObjects",
	"cmd.(*webAPIHandlers).RemoveObject",
	"cmd.(*webAPIHandlers).Login",
	"cmd.(*webAPIHandlers).GenerateAuth",
	"cmd.(*webAPIHandlers).SetAuth",
	"cmd.(*webAPIHandlers).GetAuth",
	"cmd.(*webAPIHandlers).CreateURLToken",
	"cmd.(*webAPIHandlers).Upload",
	"cmd.(*webAPIHandlers).Download",
	"cmd.(*webAPIHandlers).DownloadZip",
	"cmd.(*webAPIHandlers).GetBucketPolicy",
	"cmd.(*webAPIHandlers).ListAllBucketPolicies",
	"cmd.(*webAPIHandlers).SetBucketPolicy",
	"cmd.(*webAPIHandlers).PresignedGet",
	"cmd.(*webAPIHandlers).ServerInfo",
	"cmd.(*webAPIHandlers).StorageInfo",
	// add more here ..
}

func (level Level) String() string {
	var lvlStr string
	switch level {
	case InformationLvl:
		lvlStr = "INFO"
	case ErrorLvl:
		lvlStr = "ERROR"
	case FatalLvl:
		lvlStr = "FATAL"
	}
	return lvlStr
}

type args struct {
	Bucket   string            `json:"bucket,omitempty"`
	Object   string            `json:"object,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type traceEntry struct {
	Message   string            `json:"message,omitempty"`
	Source    []string          `json:"source,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

type api struct {
	Name string `json:"name,omitempty"`
	Args *args  `json:"args,omitempty"`
}

// LogEntry - defines fields and values of each log entry.
type LogEntry struct {
	DeploymentID string      `json:"deploymentid,omitempty"`
	Level        string      `json:"level"`
	Time         string      `json:"time"`
	API          *api        `json:"api,omitempty"`
	RemoteHost   string      `json:"remotehost,omitempty"`
	RequestID    string      `json:"requestID,omitempty"`
	UserAgent    string      `json:"userAgent,omitempty"`
	Message      string      `json:"message,omitempty"`
	Trace        *traceEntry `json:"error,omitempty"`
}

// quietFlag: Hide startup messages if enabled
// jsonFlag: Display in JSON format, if enabled
var (
	quietFlag, jsonFlag bool
	// Custom function to format error
	errorFmtFunc func(string, error, bool) string
)

// EnableQuiet - turns quiet option on.
func EnableQuiet() {
	quietFlag = true
}

// EnableJSON - outputs logs in json format.
func EnableJSON() {
	jsonFlag = true
	quietFlag = true
}

// IsJSON - returns true if jsonFlag is true
func IsJSON() bool {
	return jsonFlag
}

// IsQuiet - returns true if quietFlag is true
func IsQuiet() bool {
	return quietFlag
}

// RegisterUIError registers the specified rendering function. This latter
// will be called for a pretty rendering of fatal errors.
func RegisterUIError(f func(string, error, bool) string) {
	errorFmtFunc = f
}

func uniqueEntries(paths []string) []string {
	found := map[string]bool{}
	unqiue := []string{}

	for v := range paths {
		if _, ok := found[paths[v]]; !ok {
			found[paths[v]] = true
			unqiue = append(unqiue, paths[v])
		}
	}
	return unqiue
}

// Init sets the trimStrings to possible GOPATHs
// and GOROOT directories. Also append github.com/minio/minio
// This is done to clean up the filename, when stack trace is
// displayed when an error happens.
func Init(goPath string, goRoot string) {

	var goPathList []string
	var goRootList []string
	var defaultgoPathList []string
	var defaultgoRootList []string
	pathSeperator := ":"
	// Add all possible GOPATH paths into trimStrings
	// Split GOPATH depending on the OS type
	if runtime.GOOS == "windows" {
		pathSeperator = ";"
	}

	goPathList = strings.Split(goPath, pathSeperator)
	goRootList = strings.Split(goRoot, pathSeperator)
	defaultgoPathList = strings.Split(build.Default.GOPATH, pathSeperator)
	defaultgoRootList = strings.Split(build.Default.GOROOT, pathSeperator)

	// Add trim string "{GOROOT}/src/" into trimStrings
	trimStrings = []string{filepath.Join(runtime.GOROOT(), "src") + string(filepath.Separator)}

	// Add all possible path from GOPATH=path1:path2...:pathN
	// as "{path#}/src/" into trimStrings
	for _, goPathString := range goPathList {
		trimStrings = append(trimStrings, filepath.Join(goPathString, "src")+string(filepath.Separator))
	}

	for _, goRootString := range goRootList {
		trimStrings = append(trimStrings, filepath.Join(goRootString, "src")+string(filepath.Separator))
	}

	for _, defaultgoPathString := range defaultgoPathList {
		trimStrings = append(trimStrings, filepath.Join(defaultgoPathString, "src")+string(filepath.Separator))
	}

	for _, defaultgoRootString := range defaultgoRootList {
		trimStrings = append(trimStrings, filepath.Join(defaultgoRootString, "src")+string(filepath.Separator))
	}

	// Remove duplicate entries.
	trimStrings = uniqueEntries(trimStrings)

	// Add "github.com/minio/minio" as the last to cover
	// paths like "{GOROOT}/src/github.com/minio/minio"
	// and "{GOPATH}/src/github.com/minio/minio"
	trimStrings = append(trimStrings, filepath.Join("github.com", "minio", "minio")+string(filepath.Separator))
}

func trimTrace(f string) string {
	for _, trimString := range trimStrings {
		f = strings.TrimPrefix(filepath.ToSlash(f), filepath.ToSlash(trimString))
	}
	return filepath.FromSlash(f)
}

func getSource(level int) string {
	pc, file, lineNumber, ok := runtime.Caller(level)
	if ok {
		// Clean up the common prefixes
		file = trimTrace(file)
		_, funcName := filepath.Split(runtime.FuncForPC(pc).Name())
		return fmt.Sprintf("%v:%v:%v()", file, lineNumber, funcName)
	}
	return ""
}

// getTrace method - creates and returns stack trace
func getTrace(traceLevel int) []string {
	var trace []string
	pc, file, lineNumber, ok := runtime.Caller(traceLevel)

	for ok && file != "" {
		// Clean up the common prefixes
		file = trimTrace(file)
		// Get the function name
		_, funcName := filepath.Split(runtime.FuncForPC(pc).Name())
		// Skip duplicate traces that start with file name, "<autogenerated>"
		// and also skip traces with function name that starts with "runtime."
		if !strings.HasPrefix(file, "<autogenerated>") &&
			!strings.HasPrefix(funcName, "runtime.") {
			// Form and append a line of stack trace into a
			// collection, 'trace', to build full stack trace
			trace = append(trace, fmt.Sprintf("%v:%v:%v()", file, lineNumber, funcName))

			// Ignore trace logs beyond the following conditions
			for _, name := range matchingFuncNames {
				if funcName == name {
					return trace
				}
			}
		}
		traceLevel++
		// Read stack trace information from PC
		pc, file, lineNumber, ok = runtime.Caller(traceLevel)
	}
	return trace
}

// LogAlwaysIf prints a detailed error message during
// the execution of the server.
func LogAlwaysIf(ctx context.Context, err error) {
	if err == nil {
		return
	}

	logIf(ctx, err)
}

// LogIf prints a detailed error message during
// the execution of the server, if it is not an
// ignored error.
func LogIf(ctx context.Context, err error) {
	if err == nil {
		return
	}

	if err.Error() != diskNotFoundError {
		logIf(ctx, err)
	}
}

// logIf prints a detailed error message during
// the execution of the server.
func logIf(ctx context.Context, err error) {
	if Disable {
		return
	}

	req := GetReqInfo(ctx)

	if req == nil {
		req = &ReqInfo{API: "SYSTEM"}
	}

	API := "SYSTEM"
	if req.API != "" {
		API = req.API
	}

	tags := make(map[string]string)
	for _, entry := range req.GetTags() {
		tags[entry.Key] = entry.Val
	}

	// Get full stack trace
	trace := getTrace(3)

	// Get the cause for the Error
	message := err.Error()

	entry := LogEntry{
		DeploymentID: req.DeploymentID,
		Level:        ErrorLvl.String(),
		RemoteHost:   req.RemoteHost,
		RequestID:    req.RequestID,
		UserAgent:    req.UserAgent,
		Time:         time.Now().UTC().Format(time.RFC3339Nano),
		API:          &api{Name: API, Args: &args{Bucket: req.BucketName, Object: req.ObjectName}},
		Trace:        &traceEntry{Message: message, Source: trace, Variables: tags},
	}

	// Iterate over all logger targets to send the log entry
	for _, t := range Targets {
		t.Send(entry)
	}
}

// ErrCritical is the value panic'd whenever CriticalIf is called.
var ErrCritical struct{}

// CriticalIf logs the provided error on the console. It fails the
// current go-routine by causing a `panic(ErrCritical)`.
func CriticalIf(ctx context.Context, err error) {
	if err != nil {
		LogIf(ctx, err)
		panic(ErrCritical)
	}
}

// FatalIf is similar to Fatal() but it ignores passed nil error
func FatalIf(err error, msg string, data ...interface{}) {
	if err == nil {
		return
	}
	fatal(err, msg, data...)
}
