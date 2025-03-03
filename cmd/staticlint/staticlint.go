// Package main утилита для статического анализа кода с заранее выбранными анализаторами.
//
// # Обзор
//
// Этот мультичекер включает:
//   - Стандартные анализаторы из пакета golang.org/x/tools/go/analysis/passes
//   - Все анализаторы класса SA из пакета staticcheck.io
//   - Выбранные анализаторы из других классов пакета staticcheck.io
//   - Собственные анализаторы (noexit, errcheck)
//   - Сторонние анализаторы (exhaustive, bodyclose)
//
// # Использование
//
// Запустите мультичекер с помощью:
//
//	go run cmd/staticlint/staticlint.go [пакеты]
//
// Или скомпилируйте и запустите:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint [пакеты]
//
// # Конфигурация
//
// Мультичекер можно настроить с помощью JSON-файла, расположенного по пути cmd/staticlint/config.json.
// Файл конфигурации позволяет указать, какие анализаторы из staticcheck и stylecheck использовать.
//
// Пример конфигурации:
//
//	{
//	    "staticcheck": ["SA4006", "SA5000"],
//	    "stylecheck": ["ST1000"]
//	}
//
// Если файл конфигурации не найден или содержит ошибки, будут использованы все доступные анализаторы.
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
	"github.com/maynagashev/go-metrics/cmd/staticlint/passes/noexit"
	"github.com/nishanths/exhaustive"
	"github.com/timakin/bodyclose/passes/bodyclose"
)

// Config — имя файла конфигурации.
// Файл должен находиться в директории cmd/staticlint.
const Config = `cmd/staticlint/config.json`

// ConfigData описывает структуру файла конфигурации.
// Содержит списки анализаторов из staticcheck и stylecheck, которые нужно использовать.
type ConfigData struct {
	// Staticcheck содержит список имен анализаторов из пакета staticcheck.
	Staticcheck []string `json:"staticcheck"`
	// Stylecheck содержит список имен анализаторов из пакета stylecheck.
	Stylecheck []string `json:"stylecheck"`
}

// loadConfig загружает конфигурацию из файла.
// Возвращает конфигурацию и флаг успешности загрузки.
// Если файл не найден или содержит ошибки, возвращает пустую конфигурацию и false.
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
//
// Параметры:
//   - mychecks: текущий список анализаторов
//   - cfg: конфигурация, содержащая списки анализаторов
//   - analyzerType: тип анализаторов ("staticcheck" или "stylecheck")
//   - analyzers: полный список доступных анализаторов указанного типа
//
// Возвращает обновленный список анализаторов.
// Если в конфигурации указаны конкретные анализаторы, добавляет только их.
// Если список пуст или отсутствует, добавляет все доступные анализаторы.
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

// getAllStaticcheckAnalyzers возвращает список всех анализаторов из пакета staticcheck.
// Эти анализаторы проверяют корректность кода и выявляют потенциальные ошибки.
func getAllStaticcheckAnalyzers() []*analysis.Analyzer {
	result := make([]*analysis.Analyzer, len(staticcheck.Analyzers))
	for i, a := range staticcheck.Analyzers {
		result[i] = a.Analyzer
	}
	return result
}

// getAllStylecheckAnalyzers возвращает список всех анализаторов из пакета stylecheck.
// Эти анализаторы проверяют стиль кода и соответствие стандартам оформления.
func getAllStylecheckAnalyzers() []*analysis.Analyzer {
	result := make([]*analysis.Analyzer, len(stylecheck.Analyzers))
	for i, a := range stylecheck.Analyzers {
		result[i] = a.Analyzer
	}
	return result
}

// printAnalyzersList выводит список анализаторов с их описаниями.
// Для каждого анализатора выводится его имя и краткое описание (первая строка документации).
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

// 5. Запускает multichecker.
func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	// Инициализируем базовый набор анализаторов
	mychecks := []*analysis.Analyzer{
		// анализаторы из golang.org/x/tools/go/analysis/passes
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,

		// собственные публичные анализаторы
		errcheck.Analyzer,
		noexit.Analyzer,

		// exhaustive - проверяет полноту switch для перечислений
		exhaustive.Analyzer,

		// bodyclose - проверяет, что тела HTTP-ответов закрываются
		bodyclose.Analyzer,
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
