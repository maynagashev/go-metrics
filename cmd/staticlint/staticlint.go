package main

import (
	"encoding/json"
	"os"

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
	mychecks := []*analysis.Analyzer{
		errcheck.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
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
			for _, v := range staticcheck.Analyzers {
				if checks[v.Analyzer.Name] {
					mychecks = append(mychecks, v.Analyzer)
				}
			}
		}
	}

	multichecker.Main(
		mychecks...,
	)
}
