package noexit_test

import (
	"testing"

	"github.com/maynagashev/go-metrics/cmd/staticlint/passes/noexit"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer запускает анализатор noexit на тестовых файлах в директории testdata.
// Он проверяет, что анализатор корректно идентифицирует прямые вызовы os.Exit в функции main
// пакета main, игнорируя вызовы в других функциях или пакетах.
//
// Тест использует следующие тестовые случаи:
// - a/a.go: Содержит прямой вызов os.Exit в функции main пакета main (должен сообщить об ошибке)
// - a/b/b.go: Содержит вызов os.Exit в функции с именем main, но в другом пакете (не должен сообщать об ошибке)
//
// Каждый тестовый файл содержит комментарии с аннотациями "want", указывающими ожидаемые диагностические сообщения.
func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, noexit.NewAnalyzer(), "a")
}
