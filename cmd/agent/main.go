// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"fmt"
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
)

func main() {
	err := parseFlags()
	if err != nil {
		fmt.Printf("error parsing flags or env: %s\n", err)
	}

	serverURL := "http://" + flagServerAddr
	pollInterval := time.Duration(flagPollInterval) * time.Second
	reportInterval := time.Duration(flagReportInterval) * time.Second

	a := agent.New(serverURL, pollInterval, reportInterval)
	a.Run()
}
