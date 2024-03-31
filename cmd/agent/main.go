// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"github.com/maynagashev/go-metrics/internal/agent"
	"time"
)

func main() {
	parseFlags()

	serverURL := "http://" + flagServerAddr
	pollInterval := time.Duration(flagPollInterval) * time.Second
	reportInterval := time.Duration(flagReportInterval) * time.Second

	a := agent.New(serverURL, pollInterval, reportInterval)
	a.Run()
}
