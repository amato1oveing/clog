package log

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var logFile *os.File

func init() {
	fmt.Println("init log serve...")
	var err error
	logFile, err = os.OpenFile("./zc.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("open log file failed, err:", err)
		return
	}
	fmt.Println("init log success!")
}

type LEVEL int //日志等级
const (
	DEBUG LEVEL = iota
	INFO
	WARN
	ERROR
)

type COLOR int //日志颜色
const (
	COLOR_WHITE  = COLOR(37) // 白
	COLOR_GREEN  = COLOR(32) // 绿
	COLOR_YELLOW = COLOR(33) // 黄
	COLOR_RED    = COLOR(31) // 红
)

type Logger struct {
	LogLever LEVEL
}

func NewLogger(level string) Logger {
	return Logger{LogLever: parse(level)}
}

func parse(str string) LEVEL {
	switch strings.ToLower(str) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return DEBUG 	//默认日志等级为DEBUG
	}
}

func (this Logger) checkLevel(level LEVEL) bool {
	if this.LogLever <= level {
		return false
	}
	return true
}

func (this Logger) DEBUG(msg string, args ...interface{}) {
	if this.checkLevel(DEBUG) {
		return
	}
	for _, arg := range args {
		msg = strings.Replace(msg, "%s", inter2str(arg), 1)
	}
	msg = strings.Replace(msg, "%s", "%%s", -1)

	pc,file,line,ok:=runtime.Caller(1)
	if !ok{
		return
	}
	funcName:=runtime.FuncForPC(pc).Name()
	msg = fmt.Sprintf("[DEBUG] <%s> %s:%d:%s %s", time.Now().Format("2006-01-02-15 04:05"),file,line,funcName, msg)
	//logFile.WriteString(str)
	fmt.Fprintln(logFile, msg)
	fmt.Println(msg)
}

func (this Logger) INFO(msg string, args ...interface{}) {
	if this.checkLevel(INFO) {
		return
	}
	for _, arg := range args {
		msg = strings.Replace(msg, "%s", inter2str(arg), 1)
	}
	msg = strings.Replace(msg, "%s", "%%s", -1)
	msg = fmt.Sprintf("[INFO] <%s> %s", time.Now().Format("2006-01-02-15 04:05"), msg)
	//logFile.WriteString(msg)
	fmt.Fprintln(logFile, msg)
	fmt.Println(msg)
}

func (this Logger) WARN(msg string, args ...interface{}) {
	if this.checkLevel(WARN) {
		return
	}
	for _, arg := range args {
		msg = strings.Replace(msg, "%s", inter2str(arg), 1)
	}
	msg = strings.Replace(msg, "%s", "%%s", -1)
	msg = fmt.Sprintf("[WARN] <%s> %s", time.Now().Format("2006-01-02-15 04:05"), msg)
	//logFile.WriteString(str)
	fmt.Fprintln(logFile, msg)
	fmt.Println(msg)
}

func (this Logger) ERROR(msg string, args ...interface{}) {
	if this.checkLevel(ERROR) {
		return
	}
	for _, arg := range args {
		msg = strings.Replace(msg, "%s", inter2str(arg), 1)
	}
	msg = strings.Replace(msg, "%s", "%%s", -1)
	msg = fmt.Sprintf("[ERROR] <%s> %s", time.Now().Format("2006-01-02-15 04:05"), msg)
	//logFile.WriteString(str)
	fmt.Fprintln(logFile, msg)
	fmt.Println(msg)
}

//将interface转为string
func inter2str(value interface{}) string {
	var key string
	if value == nil {
		return key
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		key = strconv.Itoa(it)
	case uint:
		it := value.(uint)
		key = strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		key = strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		key = strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		key = strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		key = strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		key = strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		key = strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		key = strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		key = strconv.FormatUint(it, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}

	return key
}
