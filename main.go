package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/papertrail/remote_syslog2/syslog"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var host string
var port int
var useTcp bool
var format string
var useColor bool
var protocol = "udp"
var exclude string
var excludeList []string

const (
	FMT_COLOR = "\033[1;33;40mPID\033[0m: {{.PID}}; \033[1;33;40mUnit\033[0m: {{.Unit}}; \033[1;33;40mImage\033[0m: {{.Exe}}; \033[1;33;40mMessage\033[0m: {{.Message}}"
	FMT_PLAIN = "PID: {{.PID}}; Unit: {{.Unit}}; Image: {{.Exe}}; Message: {{.Message}}"
)

func init() {
	flag.StringVar(&host, "host", "", "Server to forward logs to")
	flag.IntVar(&port, "port", 0, "Port to forward logs to")
	flag.StringVar(&format, "format", FMT_PLAIN, "Format for the message")
	flag.BoolVar(&useColor, "color", false, "Use the default formatter with ANSI colors")
	flag.BoolVar(&useTcp, "useTcp", false, "Forward logs over tcp?")
	flag.StringVar(&exclude, "exclude", "", "A comma seperated list of processes to ignore. Example \"docker,sudo\" ")
	flag.Parse()
	if useTcp {
		protocol = "tcp"
	}
	if format == FMT_PLAIN && useColor == true {
		format = FMT_COLOR
	}
	if host == "" {
		fmt.Println("-host is a required argument")
		flag.Usage()
		os.Exit(1)
	}
	if port == 0 {
		fmt.Println("-port is a required argument")
		flag.Usage()
		os.Exit(1)
	}
	excludeList = strings.Split(exclude, ",")
}

type JournalMessage struct {
	Message   string `json:"MESSAGE"`
	Priority  string `json:"PRIORITY"`
	Facility  string `json:"SYSLOG_FACILITY"`
	Tag       string `json:"_COMM"`
	BootId    string `json:"_BOOT_ID"`
	Exe       string `json:"_EXE"`
	Gid       string `json:"_GID"`
	HostName  string `json:"_HOSTNAME"`
	MachineId string `json:"_MACHINE_ID"`
	PID       string `json:"_PID"`
	Unit      string `json:"_SYSTEMD_UNIT"`
	Transport string `json:"_TRANSPORT"`
	UID       string `json:"_UID"`
	Timestamp string `json:"__REALTIME_TIMESTAMP"`
}

func shouldSend(m *JournalMessage) bool {
	for _, i := range excludeList {
		if m.Tag == i {
			return false
		}
	}
	return true
}

func tailJournal(logger *syslog.Logger, t *template.Template) {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadBytes('\n')

		if err != nil {
			if err == io.EOF {
				os.Exit(0)
			}
			log.Fatalf("Stdin Error: %s", err)
		}

		data := JournalMessage{}
		err = json.Unmarshal(line, &data)
		if err != nil {
			logger.Errors <- err
			continue
		}

		if shouldSend(&data) {
			buf := new(bytes.Buffer)
			err = t.Execute(buf, data)
			if err != nil {
				logger.Errors <- err
				continue
			}
			facility, err := strconv.ParseInt(data.Facility, 0, 64)
			if err != nil {
				logger.Errors <- err
				continue
			}
			priority, err := strconv.ParseInt(data.Priority, 0, 64)
			if err != nil {
				logger.Errors <- err
				continue
			}
			timestamp, err := strconv.ParseInt(data.Timestamp, 0, 64)
			if err != nil {
				logger.Errors <- err
				continue
			}
			logger.Packets <- syslog.Packet{
				Severity: syslog.Priority(priority),
				Facility: syslog.Priority(facility),
				Time:     time.Unix(timestamp/1000000, 0).UTC(),
				Hostname: data.HostName,
				Tag:      data.Tag,
				Message:  buf.String(),
			}
		}
	}
}

func main() {
	raddr := net.JoinHostPort(host, strconv.Itoa(port))
	log.Printf("Connecting to %s over %s", raddr, protocol)
	logger, err := syslog.Dial(host, protocol, raddr, nil)

	if err != nil {
		log.Fatalf("Critical Error: %s", err)
	}

	t := template.New("Format")
	t, err = t.Parse(format)
	if err != nil {
		log.Fatalf("Critical Error: %s", err)
	}

	go tailJournal(logger, t)

	for err = range logger.Errors {
		log.Printf("Syslog error: %v", err)
	}
}
