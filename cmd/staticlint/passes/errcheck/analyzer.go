package errcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// getErrorType возвращает интерфейс типа error.
func getErrorType() *types.Interface {
	err := types.Universe.Lookup("error")
	if err == nil {
		panic("error type not found in universe")
	}
	typ := err.Type()
	if typ == nil {
		panic("error type is nil")
	}
	underlying := typ.Underlying()
	if underlying == nil {
		panic("error underlying type is nil")
	}
	iface, ok := underlying.(*types.Interface)
	if !ok {
		panic("error type is not an interface")
	}
	return iface
}

// NewAnalyzer создает новый анализатор для проверки обработки ошибок.
func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "errcheck",
		Doc:  "check for unchecked errors",
		Run:  run,
	}
}

//nolint:gochecknoglobals // Analyzer должен быть глобальной переменной по дизайну пакета analysis
var Analyzer = NewAnalyzer()

func isErrorType(t types.Type) bool {
	return types.Implements(t, getErrorType())
}

// analysisResult содержит результаты анализа.
type analysisResult struct {
	hasIssues bool
	pass      *analysis.Pass
}

// ignoredFunctions содержит имена функций, ошибки которых можно игнорировать.
//
//nolint:gochecknoglobals // Необходимо для хранения списка игнорируемых функций
var ignoredFunctions = map[string]bool{
	"fmt.Print":   true,
	"fmt.Printf":  true,
	"fmt.Println": true,
}

// shouldIgnoreCall проверяет, нужно ли игнорировать ошибки от данного вызова.
func shouldIgnoreCall(_ *analysis.Pass, call *ast.CallExpr) bool {
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkgName, ok := fun.X.(*ast.Ident)
	if !ok {
		return false
	}

	// Получаем полное имя функции в формате "пакет.функция"
	fullName := pkgName.Name + "." + fun.Sel.Name

	// Проверяем, есть ли функция в списке игнорируемых
	for prefix := range ignoredFunctions {
		if strings.HasPrefix(fullName, prefix) {
			return true
		}
	}

	return false
}

// processExprStmt проверяет выражения на необработанные ошибки.
func (r *analysisResult) processExprStmt(x *ast.ExprStmt) {
	call, ok := x.X.(*ast.CallExpr)
	if !ok {
		return
	}
	if isReturnError(r.pass, call) && !shouldIgnoreCall(r.pass, call) {
		r.hasIssues = true
		r.pass.Reportf(x.Pos(), "expression returns unchecked error")
	}
}

// processTupleAssign проверяет присваивания с множественными значениями.
func (r *analysisResult) processTupleAssign(x *ast.AssignStmt) {
	call, ok := x.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}
	if shouldIgnoreCall(r.pass, call) {
		return
	}
	results := resultErrors(r.pass, call)
	for i := range x.Lhs {
		if id, isIdent := x.Lhs[i].(*ast.Ident); isIdent && id.Name == "_" && results[i] {
			r.hasIssues = true
			r.pass.Reportf(id.NamePos, "assignment with unchecked error")
		}
	}
}

// processMultiAssign проверяет множественные присваивания.
func (r *analysisResult) processMultiAssign(x *ast.AssignStmt) {
	for i := range x.Lhs {
		id, isIdent := x.Lhs[i].(*ast.Ident)
		if !isIdent {
			continue
		}
		call, isCall := x.Rhs[i].(*ast.CallExpr)
		if !isCall {
			continue
		}
		if shouldIgnoreCall(r.pass, call) {
			continue
		}
		if id.Name == "_" && isReturnError(r.pass, call) {
			r.hasIssues = true
			r.pass.Reportf(id.NamePos, "assignment with unchecked error")
		}
	}
}

// processNode обрабатывает один узел AST.
func (r *analysisResult) processNode(node ast.Node) bool {
	switch x := node.(type) {
	case *ast.ExprStmt:
		r.processExprStmt(x)
	case *ast.AssignStmt:
		if len(x.Rhs) == 1 {
			r.processTupleAssign(x)
		} else {
			r.processMultiAssign(x)
		}
	}
	return true
}

func run(pass *analysis.Pass) (interface{}, error) {
	result := &analysisResult{
		pass: pass,
	}

	// Проходим по всем файлам в пакете
	for _, file := range pass.Files {
		// Проходим по всем узлам в файле
		ast.Inspect(file, result.processNode)
	}

	//nolint:nilnil // Стандартное поведение для анализаторов - возвращать nil, nil если проблем не найдено
	return nil, nil
}

// resultErrors возвращает булев массив со значениями true,
// если тип i-го возвращаемого значения соответствует ошибке.
func resultErrors(pass *analysis.Pass, call *ast.CallExpr) []bool {
	switch t := pass.TypesInfo.Types[call].Type.(type) {
	case *types.Named:
		return []bool{isErrorType(t)}
	case *types.Pointer:
		return []bool{isErrorType(t)}
	case *types.Tuple:
		s := make([]bool, t.Len())
		for i := range t.Len() {
			switch mt := t.At(i).Type().(type) {
			case *types.Named:
				s[i] = isErrorType(mt)
			case *types.Pointer:
				s[i] = isErrorType(mt)
			}
		}
		return s
	}
	return []bool{false}
}

// isReturnError возвращает true, если среди возвращаемых значений есть ошибка.
func isReturnError(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, isError := range resultErrors(pass, call) {
		if isError {
			return true
		}
	}
	return false
}
