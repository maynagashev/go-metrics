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

// loadConfig загружает конфигурацию из файла.
// Возвращает конфигурацию и флаг успешности загрузки.
func loadConfig(path string) (ConfigData, bool) {
	var cfg ConfigData
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println("Конфигурационный файл не найден")
		return cfg, false
	}

	// Читаем в структуру
	if err = json.Unmarshal(data, &cfg); err != nil {
		log.Println("Ошибка при чтении конфигурационного файла:", err)
		return cfg, false
	}

	return cfg, true
}

// addAnalyzers добавляет анализаторы из указанного списка в mychecks.
// Функция обрабатывает анализаторы из staticcheck и stylecheck.
func addAnalyzers(
	mychecks []*analysis.Analyzer,
	cfg ConfigData,
	analyzerType string,
	analyzers []*analysis.Analyzer,
) []*analysis.Analyzer {
	var isSet bool
	var configList []string

	// определяем существует ли секция конфигурации и список не пуст
	if analyzerType == "staticcheck" {
		isSet = len(cfg.Staticcheck) > 0
		configList = cfg.Staticcheck
	} else if analyzerType == "stylecheck" {
		isSet = len(cfg.Stylecheck) > 0
		configList = cfg.Stylecheck
	}

	// Если секция конфигурации существует и список не пуст
	if isSet {
		// Создаем карту для быстрого поиска
		checks := make(map[string]bool)
		for _, name := range configList {
			checks[name] = true
		}

		// Добавляем только указанные анализаторы
		for _, analyzer := range analyzers {
			if checks[analyzer.Name] {
				mychecks = append(mychecks, analyzer)
			}
		}
		log.Printf("Используются анализаторы %s из конфигурационного файла: %v\n", analyzerType, configList)
	} else {
		// Добавляем все анализаторы если секция не существует или пустая
		log.Printf("Используются все анализаторы %s (%s)\n", analyzerType, "секция отсутствует в конфигурации")
		mychecks = append(mychecks, analyzers...)
	}

	return mychecks
}

// getAllStaticcheckAnalyzers возвращает список анализаторов из staticcheck.
func getAllStaticcheckAnalyzers() []*analysis.Analyzer {
	result := make([]*analysis.Analyzer, len(staticcheck.Analyzers))
	for i, a := range staticcheck.Analyzers {
		result[i] = a.Analyzer
	}
	return result
}

// getAllStylecheckAnalyzers возвращает список анализаторов из stylecheck.
func getAllStylecheckAnalyzers() []*analysis.Analyzer {
	result := make([]*analysis.Analyzer, len(stylecheck.Analyzers))
	for i, a := range stylecheck.Analyzers {
		result[i] = a.Analyzer
	}
	return result
}

// printAnalyzersList выводит список анализаторов с их описаниями.
func printAnalyzersList(analyzers []*analysis.Analyzer) {
	log.Println("Итоговый список анализаторов:")
	for i, analyzer := range analyzers {
		description := analyzer.Doc
		// берем только первую строку описания
		if newlineIndex := strings.Index(description, "\n"); newlineIndex != -1 {
			description = description[:newlineIndex]
		}
		log.Printf("%d. %s: %s\n", i+1, analyzer.Name, description)
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	// Инициализируем базовый набор анализаторов
	mychecks := []*analysis.Analyzer{
		// анализаторы из golang.org/x/tools/go/analysis/passes
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,

		// собственный публичный анализатор
		errcheck.Analyzer,
	}

	// Загружаем конфигурацию
	cfg, _ := loadConfig(Config)

	// Добавляем анализаторы из staticcheck
	mychecks = addAnalyzers(mychecks, cfg, "staticcheck", getAllStaticcheckAnalyzers())

	// Добавляем анализаторы из stylecheck
	mychecks = addAnalyzers(mychecks, cfg, "stylecheck", getAllStylecheckAnalyzers())

	// Выводим список анализаторов
	printAnalyzersList(mychecks)

	// Запускаем мультичекер
	log.Println("Запуск мультичекера...")
	multichecker.Main(
		mychecks...,
	)
	log.Println("Анализ завершен.")
}
