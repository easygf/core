package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileLogger struct {
	f                 *os.File
	base              string
	curHour           string
	curHourTs         int64
	curSecondTs       int64
	maxFileSize       int64
	full              bool
	lastCheckFileSize int64
	bufChan           chan string
	flushChan         chan bool

	flushWorkerExit   bool
	flushWorkerExitMu sync.Mutex
}

func InitFileLog(dir string, name string) error {
	if logger != nil {
		return errors.New("logger has init")
	}
	l := &FileLogger{}
	err := l.Init(dir, name)
	if err != nil {
		return err
	}
	logger = l
	return nil
}

func (l *FileLogger) getFlushWorkerExit() bool {
	l.flushWorkerExitMu.Lock()
	res := l.flushWorkerExit
	l.flushWorkerExitMu.Unlock()
	return res
}

func (l *FileLogger) flushWorker() {
	for {
		select {
		case buf := <-l.bufChan:
			b := bytes.NewBufferString(buf)
			for {
				select {
				case more := <-l.bufChan:
					b.WriteString(more)
					if b.Len() >= 2*1024*1024 {
						goto OUT
					}
				default:
					goto OUT
				}
			}
		OUT:
			_ = l.realWrite(b.String())
		case <-l.flushChan:
			con := true
			for con {
				select {
				case buf := <-l.bufChan:
					_ = l.realWrite(buf)
				default:
					con = false
					break
				}
			}
			l.flushWorkerExitMu.Lock()
			l.flushWorkerExit = true
			l.flushWorkerExitMu.Unlock()
			return
		}
	}
}

func (l *FileLogger) Init(dir string, name string) error {
	if dir == "" {
		dir = "."
	}
	if !strings.HasPrefix(dir, ".") {
		// make dir
		old := UMask(0)
		defer UMask(old)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Printf("make dir fail, dir %s, err %s\n", dir, err)
			return err
		}
	}
	name = removeSuffixIfMatched(name, ".log")
	l.base = fmt.Sprintf("%s%s%s", dir, fileSep, name)
	l.maxFileSize = defaultMaxFileSize
	l.bufChan = make(chan string, 100000)
	l.flushChan = make(chan bool, 100)
	go l.flushWorker()
	return nil
}

func (l *FileLogger) needOpen() bool {
	res := false
	now := time.Now()
	if l.f == nil {
		res = true
	} else if l.full {
		ts := now.Unix()
		if l.curSecondTs+60 < ts {
			l.curSecondTs = ts
			res = true
		}
	} else {
		ts := now.Unix() / 3600
		if ts != l.curHourTs {
			l.curHourTs = ts
			res = true
		}
	}
	if res {
		l.curHour = now.Format("2006010215")
	}
	return res
}

func (l *FileLogger) checkFull() {
	now := time.Now().Unix()
	last := l.lastCheckFileSize
	if last+10 < now {
		f := l.f
		if f != nil {
			l.lastCheckFileSize = now
			size, err := f.Seek(0, io.SeekEnd)
			if err == nil {
				// get current max
				dat, err := os.ReadFile(maxLogFileSizeFile)
				if err == nil && len(dat) > 0 {
					v, err := strconv.ParseInt(strings.TrimSpace(string(dat)), 10, 64)
					if err == nil && v > 0 {
						l.maxFileSize = v
					}
				}
				if size >= l.maxFileSize {
					l.full = true
				} else {
					l.full = false
				}
			}
		}
	}
}

func (l *FileLogger) Write(buf string) error {
	if log2Stdout {
		fmt.Print(buf)
	}
	select {
	case l.bufChan <- buf:
	// sent
	default:
		println("waring: buffer channel full")
		return errors.New("buffer channel full")
	}
	return nil
}

func (l *FileLogger) realWrite(buf string) error {
	if l.needOpen() {
		err := l.open()
		if err != nil {
			return err
		}
	}
	if l.f == nil {
		return errors.New("file not open")
	}
	l.checkFull()
	if l.full {
		now := time.Now()
		if lasTime.Add(time.Minute).Before(now) {
			lasTime = now
			// println(fmt.Sprintf("file has full %s", time.Now().Format("2006-01-02 15:04:05.0000")))
		}
		return errors.New("file has full")
	}
	n, err := l.f.WriteString(buf)
	if err != nil {
		println(fmt.Sprintf("err %v,%s", err, time.Now().Format("2006-01-02 15:04:05.0000")))
		return err
	}
	for n < len(buf) {
		x, err := l.f.WriteString(buf[n:])
		if err != nil {
			println(fmt.Sprintf("err %v,%s", err, time.Now().Format("2006-01-02 15:04:05.0000")))
			return err
		}
		n += x
	}
	return nil
}
func (l *FileLogger) Sync() error {
	if l.f == nil {
		return errors.New("file not open")
	}
	return l.f.Sync()
}
func (l *FileLogger) Flush() {
	select {
	case l.flushChan <- true:
	// sent
	default:
		// channel full
	}
	for i := 0; i < 100; i++ {
		if l.getFlushWorkerExit() {
			break
		}
		time.Sleep(50 * time.Microsecond)
	}
}
func (l *FileLogger) open() error {
	logPath := fmt.Sprintf("%s%s.log", l.base, l.curHour)
	old := UMask(0)
	defer UMask(old)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Printf("open file fail, path %s, err %s", logPath, err)
		return err
	}
	oldFile := l.f
	l.f = f
	if oldFile != nil {
		err = oldFile.Close()
		if err != nil {
			fmt.Printf("close old file fail, name %s, err %s\n", oldFile.Name(), err)
		}
	}
	l.full = false
	return nil
}
