// Package noexit defines an analyzer that checks for direct calls to os.Exit
// in the main function of the main package.
package noexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the analyzer for the noexit check.
var Analyzer = &analysis.Analyzer{
	Name:     "noexit",
	Doc:      "check for direct calls to os.Exit in the main function of the main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

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

// isInMainFunc проверяет, находится ли узел в функции main
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

// isOSExitCall проверяет, является ли вызов функции вызовом os.Exit
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
