package errcheck

import (
	"errors"
	"go/ast"
	"go/types"

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

// ErrNoIssuesFound возвращается, когда анализатор не нашел проблем.
var ErrNoIssuesFound = errors.New("no issues found")

// ErrAnalysisCompleted возвращается при успешном завершении анализа.
var ErrAnalysisCompleted = errors.New("analysis completed")

// analysisResult содержит результаты анализа.
type analysisResult struct {
	hasIssues bool
	pass      *analysis.Pass
}

// processExprStmt проверяет выражения на необработанные ошибки.
func (r *analysisResult) processExprStmt(x *ast.ExprStmt) {
	call, ok := x.X.(*ast.CallExpr)
	if !ok {
		return
	}
	if isReturnError(r.pass, call) {
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

	// Если нет проблем, возвращаем "ошибку" что нет проблем
	if !result.hasIssues {
		return nil, ErrNoIssuesFound
	}

	// Возвращаем ошибку что анализ завершен
	return nil, ErrAnalysisCompleted
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
