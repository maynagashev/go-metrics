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
	"honnef.co/go/tools/stylecheck"

	"github.com/maynagashev/go-metrics/cmd/staticlint/passes/errcheck"
)

// Config — имя файла конфигурации.
const Config = `cmd/staticlint/config.json`

// ConfigData описывает структуру файла конфигурации.
type ConfigData struct {
	Staticcheck []string `json:"staticcheck"`
	Stylecheck  []string `json:"stylecheck"`
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

	// Пытаемся прочитать конфигурационный файл
	data, err := os.ReadFile(Config)
	if err == nil {
		var cfg ConfigData
		if err = json.Unmarshal(data, &cfg); err == nil {
			// Добавляем анализаторы из staticcheck, которые указаны в файле конфигурации или все
			if len(cfg.Staticcheck) > 0 {
				checks := make(map[string]bool)
				for _, v := range cfg.Staticcheck {
					checks[v] = true
				}
				for _, analyzer := range staticcheck.Analyzers {
					if checks[analyzer.Analyzer.Name] {
						mychecks = append(mychecks, analyzer.Analyzer)
					}
				}
			} else {
				log.Println("Используются все анализаторы SA*** (т.к. в конфигурационном файле не указаны анализаторы staticcheck)")
				// все анализаторы из staticcheck.io
				for _, analyzer := range staticcheck.Analyzers {
					mychecks = append(mychecks, analyzer.Analyzer)
				}
			}

			// Добавляем анализаторы из stylecheck, которые указаны в файле конфигурации
			if len(cfg.Stylecheck) > 0 {
				stChecks := make(map[string]bool)
				for _, v := range cfg.Stylecheck {
					stChecks[v] = true
				}
				for _, analyzer := range stylecheck.Analyzers {
					if stChecks[analyzer.Analyzer.Name] {
						mychecks = append(mychecks, analyzer.Analyzer)
					}
				}
				log.Println("Используются анализаторы ST**** из конфигурационного файла:", cfg.Stylecheck)
			} else {
				log.Println("Используются все анализаторы ST**** (т.к. в конфигурационном файле не указаны анализаторы stylecheck)")
				for _, analyzer := range stylecheck.Analyzers {
					mychecks = append(mychecks, analyzer.Analyzer)
				}
			}
		} else {
			log.Println("Ошибка при чтении конфигурационного файла:", err)
		}
	} else {
		log.Println("Конфигурационный файл не найден, используются все анализаторы")
	}

	// выводим список анализаторов, короткий список c нумерацией и однострочным описанием
	log.Println("Итоговый список анализаторов:")
	for i, analyzer := range mychecks {
		description := analyzer.Doc
		// берем только первую строку описания
		if newlineIndex := strings.Index(description, "\n"); newlineIndex != -1 {
			description = description[:newlineIndex]
		}
		log.Printf("%d. %s: %s\n", i+1, analyzer.Name, description)
	}

	log.Println("Запуск мультичекера...")
	multichecker.Main(
		mychecks...,
	)
	log.Println("Анализ завершен.")
}
