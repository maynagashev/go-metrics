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
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer - анализатор для проверки noexit.
// Он обнаруживает прямые вызовы os.Exit в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name:     "noexit",
	Doc:      "check for direct calls to os.Exit in the main function of the main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// run реализует логику анализа для анализатора noexit.
// Он проверяет наличие прямых вызовов os.Exit в функции main пакета main.
//
// Функция выполняет следующие шаги:
// 1. Проверяет, является ли текущий пакет пакетом "main"
// 2. Использует инспектор для поиска всех вызовов функций
// 3. Для каждого вызова проверяет, находится ли он в функции main
// 4. Если вызов находится в main, проверяет, является ли он вызовом os.Exit
// 5. Сообщает об ошибке, если обнаружен прямой вызов os.Exit в main
func run(pass *analysis.Pass) (interface{}, error) {
	// Получаем инспектор из результатов работы предыдущего анализатора
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Фильтр для поиска вызовов функций
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	// Проверяем, что мы находимся в пакете main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	// Используем инспектор для поиска вызовов функций
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		// Проверяем, что вызов находится в функции main
		if !isInMainFunc(pass, call) {
			return
		}

		// Проверяем, что это вызов os.Exit
		if isOSExitCall(pass, call) {
			pass.Reportf(call.Pos(), "direct call to os.Exit in main function is prohibited")
		}
	})

	return nil, nil
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
