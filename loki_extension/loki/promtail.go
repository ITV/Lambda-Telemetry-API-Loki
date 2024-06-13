package loki

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ITV/promtail-client-1/promtail"
)

type LogEntry struct {
	Record string `json:"record"`
	Time   string `json:"time"`
	Type   string `json:"type"`
	Level  string `json:"level"`
}

var (
	loki       promtail.Client
	conf       promtail.ClientConfig
	labels     []string
	sendLabels []string
	cpLabels   []string
	err        error
)

// Init Promtail Client
func init() {
	labels = []string{"source=\"lambda\""}
	sendLabels = []string{"source = lambda"}

	// CP Standard Labels
	environment := os.Getenv("ENVIRONMENT")
	service := os.Getenv("SERVICE")
	component := os.Getenv("COMPONENT")

	cpLabels = []string{
		fmt.Sprintf("environment=\"%s\"", environment),
		fmt.Sprintf("service=\"%s\"", service),
		fmt.Sprintf("component=\"%s\"", component),
		fmt.Sprintf("job=\"%s-%s\"", service, component),
	}
	for _, element := range os.Environ() {
		if !strings.HasPrefix(element, "OTEL_LABEL_") {
			continue
		}
		v := strings.Split(strings.TrimPrefix(element, "OTEL_LABEL_"), "=")
		key := strings.ToLower(v[0])
		val := v[1]
		labels = append(labels, fmt.Sprintf("%s=\"%s\"", key, val))
		sendLabels = append(labels, fmt.Sprintf("%s = %s", key, val))
	}
	labels = append(labels, cpLabels...)

	lokiIp := os.Getenv("LOKI_URL")
	if len(lokiIp) == 0 {
		panic("LOKI Ip undefined")
	}

	conf = promtail.ClientConfig{
		PushURL:            fmt.Sprintf("%s/api/v1/push", lokiIp),
		Labels:             fmt.Sprintf("{%s}", strings.Join(labels, ",")),
		BatchWait:          5 * time.Second,
		BatchEntriesNumber: 10000,
	}
	loki, err = promtail.NewClientProto(conf)
	if err != nil {
		log.Println("Promtail init error")
		log.Println(err)
	}
}

func LokiSend(record *string) {
	var logEntry LogEntry
	level := "INFO"
	if json.Unmarshal([]byte(*record), &logEntry) == nil {
		if logEntry.Level != "" {
			level = logEntry.Level
		}
		newLabels := append(labels, fmt.Sprintf("level=\"%s\"", level))
		conf.Labels = fmt.Sprintf("{%s}", strings.Join(newLabels, ","))
		loki, err = promtail.NewClientProto(conf)
		if err != nil {
			log.Println("Promtail re-init error")
			log.Println(err)
			return
		}
	} else {
		newLabels := append(labels, fmt.Sprintf("level=\"%s\"", level))
		conf.Labels = fmt.Sprintf("{%s}", strings.Join(newLabels, ","))
		loki, err = promtail.NewClientProto(conf)
		if err != nil {
			log.Println("Promtail re-init error")
			log.Println(err)
			return
		}
	}
	loki.Infof(strings.TrimSuffix(*record, "\n"))
}

func LokiShutdown() {
	loki.Shutdown()
}
