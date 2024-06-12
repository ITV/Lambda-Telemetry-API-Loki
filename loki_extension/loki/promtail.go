package loki

import (
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
	// product := os.Getenv("PRODUCT")     - minimise labels: arguably redundant
	// ecosystem := os.Getenv("ECOSYSTEM") - minimise labels: arguably redundant
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
		SendLevel:          promtail.INFO,
		PrintLevel:         promtail.ERROR,
	}
	loki, err = promtail.NewClientProto(conf)
	if err != nil {
		log.Println("Promtail init error")
		log.Println(err)
	}
}

func LokiSend(record *string) {
	tstamp := time.Now().String()
	loki.Infof("{\"@timestamp\": \"%s\", \"message\": \"%v\"}", tstamp, strings.TrimSuffix(*record, "\n"))
}

func LokiShutdown() {
	loki.Shutdown()
}
