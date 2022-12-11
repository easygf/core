package log

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/easygf/core/log/atexit"
	"github.com/petermattis/goid"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int8

const defaultMaxFileSize int64 = 4 * 1024 * 1024 * 1024
const (
	maxLogFileSizeFile = "/etc/brick/max_log_size"
)
const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	DPanicLevel
	PanicLevel
	FatalLevel
	ImportantLevel
)
const (
	colorRed = uint8(iota + 91)
	colorGreen
	colorYellow
	colorBlue
	colorPurple
)

var colorEnd string
var red string
var green string
var yellow string
var purple string
var blue string
var EnableLogCtx = true
var logCtx = map[int64]string{}
var logCtxMu sync.RWMutex

var lasTime time.Time

func init() {
	red = fmt.Sprintf("\x1b[%dm", colorRed)
	green = fmt.Sprintf("\x1b[%dm", colorGreen)
	yellow = fmt.Sprintf("\x1b[%dm", colorYellow)
	// 这里先这样了，后面统一改颜色
	blue = fmt.Sprintf("\u001B[%d;1m", 36)
	purple = fmt.Sprintf("\x1b[%dm", colorPurple)
	colorEnd = "\x1b[0m"
	atexit.Register(func() {
		if logger != nil {
			logger.Flush()
		}
		if loggerImportant != nil {
			loggerImportant.Flush()
		}
	})
}
func SetLogCtx(c string, i int64) {
	if i == 0 {
		i = goid.Get()
	}
	logCtxMu.Lock()
	if c == "" {
		delete(logCtx, i)
	} else {
		logCtx[i] = c
	}
	logCtxMu.Unlock()
}
func GetLogCtx(i int64) string {
	if i == 0 {
		i = goid.Get()
	}
	var v string
	logCtxMu.RLock()
	v = logCtx[i]
	logCtxMu.RUnlock()
	return v
}
func Red(s string) string {
	return red + s + colorEnd
}
func Green(s string) string {
	return green + s + colorEnd
}
func Yellow(s string) string {
	return yellow + s + colorEnd
}
func Blue(s string) string {
	return blue + s + colorEnd
}
func Purple(s string) string {
	return purple + s + colorEnd
}
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case DPanicLevel:
		return "dpanic"
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	case ImportantLevel:
		return "important"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}
func (l Level) ShortString() string {
	switch l {
	case DebugLevel:
		return "DBG "
	case InfoLevel:
		return "INF "
	case WarnLevel:
		return "WAR "
	case ErrorLevel:
		return "ERR "
	case DPanicLevel:
		return "PAN "
	case PanicLevel:
		return "PAN "
	case FatalLevel:
		return "FAT "
	case ImportantLevel:
		return "IMP "
	default:
		return fmt.Sprintf("L(%d) ", l)
	}
}

func (l Level) Color() string {
	switch l {
	case DebugLevel, InfoLevel, ImportantLevel:
		return green
	case WarnLevel:
		return yellow
	default:
		return red
	}
}

var level = DebugLevel
var fileSep string
var modName = "UNKNOWN"
var logger ILogger
var DefaultLogDir string
var loggerImportant ILogger

func init() {
	if runtime.GOOS == "windows" {
		DefaultLogDir = `c:\brick\log`
	} else {
		DefaultLogDir = "/home/brick/log"
	}
}
func SetModName(name string) {
	modName = name
}
func GetModName() string {
	return modName
}
func SetLogLevel(l Level) {
	level = l
}
func SetLog2Stdout(v bool) {
	log2Stdout = v
}
func removeSuffixIfMatched(s string, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		return s[0 : len(s)-len(suffix)]
	}
	return s
}

var pid = 0
var formatTimeSec uint32
var formatTimeSecStr string

func formatTime(t time.Time) string {
	sec := uint32(t.Unix())
	pre := formatTimeSec
	preStr := formatTimeSecStr
	if pre == sec {
		// 受并行优化的影响，小概率取了旧值，因为是打LOG，就不搞这么严谨了
		return preStr
	}
	x := t.Format("01-02T15:04:05")
	formatTimeSec = sec
	formatTimeSecStr = x
	return x
}

func formatLog(opt *Optimization, l Level, buf string, callerSkip int) string {
	now := time.Now()

	var b bytes.Buffer

	// mod
	b.WriteString(modName)

	routineId := goid.Get()

	// 进程、协程
	b.WriteString(fmt.Sprintf("(%d,%d) ", pid, routineId))
	// 时间
	b.WriteString(formatTime(now))
	b.WriteString(fmt.Sprintf(".%04d ", now.Nanosecond()/100000))

	reqId := GetLogCtx(routineId)
	if reqId != "" {
		b.WriteString("<")
		b.WriteString(reqId)
		b.WriteString("> ")
	}

	// 日志级别
	b.WriteString(l.Color())
	b.WriteString(l.ShortString())

	var callerName, callerFile string
	var callerLine int
	if opt == nil || opt.CallerLine == 0 {
		var ok bool
		var pc uintptr
		pc, callerFile, callerLine, ok = runtime.Caller(callerSkip)
		callerName = ""
		if ok {
			callerName = runtime.FuncForPC(pc).Name()
		}

	} else {
		callerFile = opt.ShortFile
		callerName = opt.CallerName
		callerLine = opt.CallerLine
	}

	// 调用位置
	filePath, fileFunc := getPackageName(callerName)
	b.WriteString(path.Join(filePath, path.Base(callerFile)))
	b.WriteString(":")
	b.WriteString(fmt.Sprintf("%d:", callerLine))
	b.WriteString(fileFunc)
	b.WriteString(colorEnd)
	b.WriteString(" ")

	// 文本内容
	b.WriteString(buf)
	b.WriteString("\n")

	return b.String()
}

func getPackageName(f string) (string, string) {
	slashIndex := strings.LastIndex(f, "/")
	if slashIndex > 0 {
		idx := strings.Index(f[slashIndex:], ".") + slashIndex
		return f[:idx], f[idx+1:]
	}

	return f, ""
}

func init() {
	if runtime.GOOS == "windows" {
		fileSep = `\`
	} else {
		fileSep = "/"
	}
	pid = os.Getpid()
}

type ILogger interface {
	Write(buf string) error
	Sync() error
	Flush()
}

func initLogImportant() error {
	if loggerImportant != nil {
		return nil
	}
	l := &FileLogger{}
	err := l.Init("/home/brick/log", "")
	if err != nil {
		return err
	}
	loggerImportant = l
	return nil
}

func PrintStack(skip int) {
	for ; ; skip++ {
		pc, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		name := runtime.FuncForPC(pc)
		if name.Name() == "runtime.goexit" {
			break
		}
		Errorf("#STACK: %s %s:%d", name.Name(), file, line)
	}
}

func logIt(opt *Optimization, l Level, msg string) {
	if l < level {
		return
	}
	msg = formatLog(opt, l, msg, 4)
	if logger != nil {
		_ = logger.Write(msg)
	} else {
		fmt.Print(msg)
	}
}

func logItImportant(msg string) {
	msg = formatLog(nil, ImportantLevel, msg, 4)
	_ = loggerImportant.Write(msg)
}

func afterLog(l Level) {
	if l == FatalLevel || l == PanicLevel || l == DPanicLevel {
		PrintStack(4)
	}
	if l == FatalLevel {
		os.Exit(1)
	}
	if l == PanicLevel {
		panic("")
	}
}

type Optimization struct {
	ShortFile  string
	CallerName string
	CallerLine int
}

func logItFmt(opt *Optimization, l Level, template string, args ...interface{}) {
	msg := template
	if msg == "" && len(args) > 0 {
		msg = fmt.Sprint(args...)
	} else if msg != "" && len(args) > 0 {
		msg = fmt.Sprintf(template, args...)
	}
	logIt(opt, l, msg)
	afterLog(l)
}

func logItFmtImportant(template string, args ...interface{}) {
	if loggerImportant == nil {
		err := initLogImportant()
		if err != nil {
			return
		}
		if loggerImportant == nil {
			panic("Unreachable")
		}
	}
	msg := template
	if msg == "" && len(args) > 0 {
		msg = fmt.Sprint(args...)
	} else if msg != "" && len(args) > 0 {
		msg = fmt.Sprintf(template, args...)
	}
	logItImportant(msg)
}

func logItArgs(l Level, args ...interface{}) {
	msg := fmt.Sprint(args...)
	logIt(nil, l, msg)
	afterLog(l)
}

func logItArgsWithOpt(opt *Optimization, l Level, args ...interface{}) {
	msg := fmt.Sprint(args...)
	logIt(opt, l, msg)
	afterLog(l)
}

func ByCodef(code int, template string, args ...interface{}) {
	prefix := fmt.Sprintf("errcode %d ", code)
	if code == 0 {
		logItFmt(nil, InfoLevel, prefix+template, args...)
	} else if code > 0 {
		logItFmt(nil, WarnLevel, prefix+template, args...)
	} else {
		logItFmt(nil, ErrorLevel, prefix+template, args...)
	}
}
func Important(template string, args ...interface{}) {
	logItFmt(nil, ImportantLevel, template, args...)
	logItFmtImportant(template, args...)
}
func Infof(template string, args ...interface{}) {
	logItFmt(nil, InfoLevel, template, args...)
}
func InfofWithOpt(opt *Optimization, template string, args ...interface{}) {
	logItFmt(opt, InfoLevel, template, args...)
}
func Printf(template string, args ...interface{}) {
	logItFmt(nil, InfoLevel, template, args...)
}
func Fatal(args ...interface{}) {
	logItArgs(FatalLevel, args...)
}
func Panic(args ...interface{}) {
	logItArgs(PanicLevel, args...)
}
func DPanic(args ...interface{}) {
	logItArgs(DPanicLevel, args...)
}
func Error(args ...interface{}) {
	logItArgs(ErrorLevel, args...)
}
func ByCode(code int, args ...interface{}) {
	prefix := fmt.Sprintf("errcode %d ", code)
	args = append([]interface{}{prefix}, args...)
	if code == 0 {
		logItArgs(InfoLevel, args...)
	} else if code > 0 {
		logItArgs(WarnLevel, args...)
	} else {
		logItArgs(ErrorLevel, args...)
	}
}
func Warn(args ...interface{}) {
	logItArgs(WarnLevel, args...)
}
func Info(args ...interface{}) {
	logItArgs(InfoLevel, args...)
}
func InfoWithOpt(opt *Optimization, args ...interface{}) {
	logItArgsWithOpt(opt, InfoLevel, args...)
}
func Debug(args ...interface{}) {
	// fast check
	if DebugLevel < level {
		return
	}
	logItArgs(DebugLevel, args...)
}
func Debugf(template string, args ...interface{}) {
	// fast check
	if DebugLevel < level {
		return
	}
	logItFmt(nil, DebugLevel, template, args...)
}
func Warnf(template string, args ...interface{}) {
	logItFmt(nil, WarnLevel, template, args...)
}
func WarnfWithOpt(opt *Optimization, template string, args ...interface{}) {
	logItFmt(opt, WarnLevel, template, args...)
}
func Errorf(template string, args ...interface{}) {
	logItFmt(nil, ErrorLevel, template, args...)
}
func ErrorfWithOpt(opt *Optimization, template string, args ...interface{}) {
	logItFmt(opt, ErrorLevel, template, args...)
}
func DPanicf(template string, args ...interface{}) {
	logItFmt(nil, DPanicLevel, template, args...)
}
func Panicf(template string, args ...interface{}) {
	logItFmt(nil, PanicLevel, template, args...)
}
func Fatalf(template string, args ...interface{}) {
	logItFmt(nil, FatalLevel, template, args...)
}
func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return errors.New("logger not open")
}
