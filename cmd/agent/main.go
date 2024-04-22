// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"time"

	"github.com/maynagashev/go-metrics/internal/agent"
)

func main() {
	flags := mustParseFlags()

	serverURL := "http://" + flags.Server.Addr
	pollInterval := time.Duration(flags.Server.PollInterval) * time.Second
	reportInterval := time.Duration(flags.Server.ReportInterval) * time.Second

	a := agent.New(serverURL, pollInterval, reportInterval)
	a.Run()
}
