// Package noexit проверяет, что в функции main пакета main нет прямых вызовов os.Exit.
//
// # Обзор
//
// Анализатор noexit обнаруживает прямые вызовы os.Exit в функции main пакета main.
// Использование os.Exit напрямую в функции main может привести к проблемам, поскольку
// это немедленно завершает программу без выполнения отложенных функций и без возможности
// корректной очистки ресурсов.
//
// # Использование
//
// Чтобы использовать этот анализатор, включите его в ваш мультичекер:
//
//	mychecks := []*analysis.Analyzer{
//		noexit.Analyzer,
//		// другие анализаторы...
//	}
//	multichecker.Main(mychecks...)
//
// # Пример
//
// Следующий код вызовет предупреждение:
//
//	package main
//
//	import (
//		"fmt"
//		"os"
//	)
//
//	func main() {
//		fmt.Println("Hello, world!")
//		os.Exit(0) // Это вызовет предупреждение
//	}
//
// Лучшим подходом будет нормальный возврат из функции main или использование
// другого механизма завершения программы, который позволяет выполнить отложенные функции.
package noexit

import (
	"errors"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Константы для анализатора.
const (
	mainPackageName = "main"
	mainFuncName    = "main"
)

// ErrNotMainPackage возвращается, когда анализируемый пакет не является main.
var ErrNotMainPackage = errors.New("not a main package")

// NewAnalyzer создает новый анализатор для проверки noexit.
// Он обнаруживает прямые вызовы os.Exit в функции main пакета main.
func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "noexit",
		Doc:      "check for direct calls to os.Exit in the main function of the main package",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      run,
	}
}

// Analyzer - анализатор для проверки noexit.
// Он обнаруживает прямые вызовы os.Exit в функции main пакета main.
//
//nolint:gochecknoglobals // Analyzer должен быть глобальной переменной для доступа из других пакетов
var Analyzer = NewAnalyzer()

// 5. Сообщает об ошибке, если обнаружен прямой вызов os.Exit в main.
func run(pass *analysis.Pass) (interface{}, error) {
	// Получаем инспектор из результатов работы предыдущего анализатора
	inspectResult, ok := pass.ResultOf[inspect.Analyzer]
	if !ok {
		return nil, errors.New("inspect analyzer result not found")
	}

	inspect, ok := inspectResult.(*inspector.Inspector)
	if !ok {
		return nil, errors.New("inspect analyzer result is not of type *inspector.Inspector")
	}

	// Фильтр для поиска вызовов функций
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	// Проверяем, что мы находимся в пакете main
	if pass.Pkg.Name() != mainPackageName {
		return nil, ErrNotMainPackage
	}

	// Используем инспектор для поиска вызовов функций
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		callExpr, isCallExpr := n.(*ast.CallExpr)
		if !isCallExpr {
			return
		}

		// Проверяем, что вызов находится в функции main
		if !isInMainFunc(pass, callExpr) {
			return
		}

		// Проверяем, что это вызов os.Exit
		if isOSExitCall(pass, callExpr) {
			pass.Reportf(callExpr.Pos(), "direct call to os.Exit in main function is prohibited")
		}
	})

	return nil, nil //nolint:nilnil // Стандартное поведение для анализаторов - возвращать nil, nil если проблем не найдено
}

// isInMainFunc проверяет, находится ли узел внутри функции main.
// Он обходит AST для поиска содержащего функцию объявления
// и проверяет, является ли она функцией main.
//
// Параметры:
//   - pass: проход анализа, содержащий AST и информацию о типах
//   - node: узел AST для проверки
//
// Возвращает true, если узел находится внутри функции main, иначе false.
func isInMainFunc(pass *analysis.Pass, node ast.Node) bool {
	// Находим ближайшую функцию, содержащую узел
	var enclosingFunc *ast.FuncDecl
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if fd, ok := n.(*ast.FuncDecl); ok {
				if fd.Name.Name == "main" {
					// Проверяем, находится ли узел внутри этой функции
					if fd.Pos() <= node.Pos() && node.Pos() <= fd.End() {
						enclosingFunc = fd
						return false
					}
				}
			}
			return true
		})
		if enclosingFunc != nil {
			break
		}
	}

	return enclosingFunc != nil && enclosingFunc.Name.Name == "main"
}

// isOSExitCall проверяет, является ли вызов функции прямым вызовом os.Exit.
// Он анализирует выражение вызова, чтобы определить, вызывается ли функция Exit
// из пакета os.
//
// Параметры:
//   - pass: проход анализа, содержащий AST и информацию о типах
//   - call: выражение вызова для проверки
//
// Возвращает true, если вызов является вызовом os.Exit, иначе false.
func isOSExitCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Проверяем, что вызов имеет форму X.Y (например, os.Exit)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Получаем информацию о типе X
	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Проверяем, что X - это "os"
	obj := pass.TypesInfo.ObjectOf(x)
	if obj == nil || obj.Name() != "os" {
		return false
	}

	// Проверяем, что Y - это "Exit"
	return sel.Sel.Name == "Exit"
}
