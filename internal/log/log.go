// Package log is a console logger with colored output that can run
// synchronously or asynchronously.
//
// Usage in main:
//
//	func main() {
//		os.Exit(run())
//	}
//
//	func run() int {
//		// flushes the message queue, logs panics, resets console color
//		defer log.Defer()
//
//		// you can change Logger settings before StartGoroutine()
//		// Logger.TimeFormat = ...
//
//		// optional: enables async mode, see "Modes"
//		log.StartGoroutine()
//
//		logger := log.LoggerContext{Name: "main"}
//		logger.Info("server started")
//		// [2026-10-29 15:41:22][main/INFO] server started
//
//		return 0
//	}
//
// Don't call os.Exit in run(): deferred functions don't run on os.Exit, so
// queued messages would be lost. Return an exit code from run() instead.
//
// Modes:
//   - Sync (default): Log prints the message immediately in the calling goroutine.
//   - Async (after StartGoroutine): Log pushes the message into a buffered
//     channel and a worker goroutine prints it. Log blocks only when the
//     buffer is full.
package log

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type Severity int32

const (
	SeverityTrace Severity = iota
	SeverityDebug
	SeverityInfo
	SeverityWarn
	SeverityError
)

var _SeverityNames = [...]string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

func (s Severity) String() string {
	return _SeverityNames[int(s)]
}

type message struct {
	text  string
	color ConsoleColor
}

type AsyncLogger struct {
	TimeFormat   string
	ColorMap     map[Severity]ConsoleColor
	QueueMaxSize int // N messages to enqueue without blocking the goroutine

	messageQueue chan message
	waitWorker   sync.WaitGroup
	mutex        sync.RWMutex
	running      bool
}

var Logger = NewLogger()

func NewLogger() *AsyncLogger {
	return &AsyncLogger{
		TimeFormat: "2006-01-02 15:04:05",
		ColorMap: map[Severity]ConsoleColor{
			SeverityTrace: DarkGray,
			SeverityDebug: Magenta,
			SeverityInfo:  Gray,
			SeverityWarn:  Yellow,
			SeverityError: Red,
		},
		QueueMaxSize: 100,
	}
}

func StartGoroutine() {
	Logger.mutex.Lock()
	Logger.running = true
	Logger.mutex.Unlock()
	// init it here to allow users change Logger.QueueMaxSize before StartGoroutine()
	Logger.messageQueue = make(chan message, Logger.QueueMaxSize)
	Logger.waitWorker.Go(worker)
}

// call this as `defer log.Defer()` in `main()“
func Defer() {
	defer ResetColor()
	if r := recover(); r != nil {
		Log("main", SeverityError, "Application panicked: ", r)
		closeAndPrintRemaining()
		// re-panic after flushing logs so stack trace prints normally
		panic(r)
	} else {
		// application shutting down normally
		closeAndPrintRemaining()
	}
}

type LoggerContext struct {
	Name string
}

func (ctx *LoggerContext) Trace(a ...any) {
	Log(ctx.Name, SeverityTrace, a...)
}
func (ctx *LoggerContext) Debug(a ...any) {
	Log(ctx.Name, SeverityDebug, a...)
}
func (ctx *LoggerContext) Info(a ...any) {
	Log(ctx.Name, SeverityInfo, a...)
}
func (ctx *LoggerContext) Warn(a ...any) {
	Log(ctx.Name, SeverityWarn, a...)
}
func (ctx *LoggerContext) Error(a ...any) {
	Log(ctx.Name, SeverityError, a...)
}

func Log(context string, severity Severity, a ...any) {
	msg := buildMessage(context, severity, a...)

	Logger.mutex.RLock() // many goroutines can hold RLock at once
	defer Logger.mutex.RUnlock()
	if !Logger.running {
		// worker is stopped or hasn't start yet
		// print synchronously
		msg.print()
		return
	}

	Logger.messageQueue <- msg
}

func buildMessage(context string, severity Severity, a ...any) message {
	timeStr := time.Now().Format(Logger.TimeFormat)
	severityStr := severity.String()
	var contextAndSeverity string
	if context == "" {
		contextAndSeverity = severityStr
	} else {
		contextAndSeverity = context + "/" + severityStr
	}

	prefix := "[" + timeStr + "][" + contextAndSeverity + "] "
	text := prefix + fmt.Sprint(a...) + "\n"
	color := Logger.ColorMap[severity]

	return message{text, color}
}

func (msg message) print() {
	SetColorFg(msg.color)
	if _, err := os.Stdout.Write([]byte(msg.text)); err != nil {
		panic(err)
	}
}

func worker() {
	for msg := range Logger.messageQueue {
		msg.print()
	}
}

func closeAndPrintRemaining() {
	// waits until no Log() is mid-send
	Logger.mutex.Lock()
	if !Logger.running {
		Logger.mutex.Unlock()
		return
	}
	Logger.running = false
	close(Logger.messageQueue)
	Logger.mutex.Unlock()
	Logger.waitWorker.Wait()
}
