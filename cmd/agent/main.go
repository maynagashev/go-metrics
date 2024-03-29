// Агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP
package main

import (
	"github.com/maynagashev/go-metrics/internal/agent"
	"time"
)

func main() {
	a := agent.New("http://localhost:8080/metrics", 2*time.Second, 10*time.Second)
	a.Run()
}
