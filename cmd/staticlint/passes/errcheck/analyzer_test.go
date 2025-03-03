// Package errcheck_test provides tests for the errcheck analyzer.
// It verifies that the analyzer correctly identifies unchecked errors in Go code.
package errcheck_test

import (
	"testing"

	"github.com/maynagashev/go-metrics/cmd/staticlint/passes/errcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer запускает анализатор errcheck на тестовых файлах в директории testdata.
// Он проверяет, что анализатор корректно идентифицирует необработанные ошибки в различных сценариях:
// - Вызовы функций, которые возвращают ошибки, но ошибка не проверяется
// - Присваивания, где ошибка явно отбрасывается с помощью "_"
// - Случаи, когда ошибки от определенных функций (например, fmt.Print) можно безопасно игнорировать
//
// Тест использует фреймворк analysistest для запуска анализатора на всех пакетах
// в директории testdata. Каждый тестовый файл содержит комментарии с аннотациями "want",
// указывающими ожидаемые диагностические сообщения.
func TestAnalyzer(t *testing.T) {
	// функция analysistest.Run применяет тестируемый анализатор
	// к пакетам из папки testdata и проверяет ожидания
	// ./... — проверка всех поддиректорий в testdata
	analysistest.Run(t, analysistest.TestData(), errcheck.NewAnalyzer(), "./...")
}
