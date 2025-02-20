package errcheck_test

import (
	"testing"

	"github.com/maynagashev/go-metrics/cmd/staticlint/passes/errcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	// функция analysistest.Run применяет тестируемый анализатор
	// к пакетам из папки testdata и проверяет ожидания
	// ./... — проверка всех поддиректорий в testdata
	analysistest.Run(t, analysistest.TestData(), errcheck.Analyzer, "./...")
}
