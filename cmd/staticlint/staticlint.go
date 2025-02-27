package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"

	"github.com/maynagashev/go-metrics/cmd/staticlint/passes/errcheck"
)

// Config — имя файла конфигурации.
const Config = `config.json`

// ConfigData описывает структуру файла конфигурации.
type ConfigData struct {
	Staticcheck []string `json:"staticcheck"`
}

func main() {
	log.SetOutput(os.Stdout)

	mychecks := []*analysis.Analyzer{
		// анализаторы из golang.org/x/tools/go/analysis/passes
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,

		// собственный публичный анализатор
		errcheck.Analyzer,
	}

	// анализаторы из staticcheck.io
	for _, analyzer := range staticcheck.Analyzers {
		mychecks = append(mychecks, analyzer.Analyzer)
	}

	// Пытаемся прочитать конфигурационный файл
	data, err := os.ReadFile(Config)
	if err == nil {
		var cfg ConfigData
		if err = json.Unmarshal(data, &cfg); err == nil {
			// Добавляем анализаторы из staticcheck, которые указаны в файле конфигурации
			checks := make(map[string]bool)
			for _, v := range cfg.Staticcheck {
				checks[v] = true
			}
			for _, analyzer := range staticcheck.Analyzers {
				if checks[analyzer.Analyzer.Name] {
					mychecks = append(mychecks, analyzer.Analyzer)
				}
			}
		}
	}

	// выводим список анализаторов, короткий список c нумерацией и однострочным описанием
	log.Println("Включает в себя следующие анализаторы:")
	for i, analyzer := range mychecks {
		description := analyzer.Doc
		// берем только первую строку описания
		if newlineIndex := strings.Index(description, "\n"); newlineIndex != -1 {
			description = description[:newlineIndex]
		}
		log.Printf("%d. %s: %s\n", i+1, analyzer.Name, description)
	}

	log.Println("Запуск анализатора...")
	multichecker.Main(
		mychecks...,
	)
	log.Println("Анализ завершен.")
}
