// Package errcheck реализует анализатор для проверки необработанных ошибок в Go-коде.
// Анализатор обнаруживает случаи, когда возвращаемая функцией ошибка игнорируется
// или явно отбрасывается путем присваивания её "_".
//
// # Обзор
//
// Обработка ошибок является критически важной частью написания надежного Go-кода.
// Этот анализатор помогает убедиться, что ошибки, возвращаемые функциями,
// правильно проверяются и обрабатываются.
//
// Анализатор обнаруживает два основных паттерна:
//   - Вызов функции, которая возвращает ошибку, без использования результата
//   - Явное отбрасывание ошибки путем присваивания её "_"
//
// # Использование
//
// Чтобы использовать этот анализатор, включите его в ваш мультичекер:
//
//	mychecks := []*analysis.Analyzer{
//		errcheck.Analyzer,
//		// другие анализаторы...
//	}
//	multichecker.Main(mychecks...)
//
// # Пример
//
// Следующий код вызовет предупреждения:
//
//	func example() {
//		// Ошибка не проверяется
//		os.Remove("file.txt")
//
//		// Ошибка явно отбрасывается
//		_, _ = os.Open("file.txt")
//	}
//
// Лучшим подходом будет правильная обработка ошибок:
//
//	func example() error {
//		err := os.Remove("file.txt")
//		if err != nil {
//			return fmt.Errorf("failed to remove file: %w", err)
//		}
//
//		file, err := os.Open("file.txt")
//		if err != nil {
//			return fmt.Errorf("failed to open file: %w", err)
//		}
//		defer file.Close()
//		return nil
//	}
package errcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// getErrorType возвращает интерфейс типа error.
// Функция получает тип error из universe scope и возвращает его как интерфейс.
// Паникует, если тип error не найден или имеет неожиданный тип.
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
// Возвращает настроенный анализатор, готовый к использованию.
func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "errcheck",
		Doc:  "check for unchecked errors",
		Run:  run,
	}
}

// Analyzer - анализатор для проверки необработанных ошибок.
// Он обнаруживает случаи, когда возвращаемая функцией ошибка игнорируется или явно отбрасывается.
//
//nolint:gochecknoglobals // Analyzer должен быть глобальной переменной для доступа из других пакетов
var Analyzer = NewAnalyzer()

func isErrorType(t types.Type) bool {
	return types.Implements(t, getErrorType())
}

// analysisResult содержит результаты анализа.
// Он отслеживает, были ли найдены проблемы, и хранит ссылку на проход анализа.
type analysisResult struct {
	hasIssues bool
	pass      *analysis.Pass
}

// getIgnoredFunctions возвращает карту имен функций, ошибки которых можно игнорировать.
// Это карта для эффективного поиска, с именами функций в качестве ключей.
func getIgnoredFunctions() map[string]bool {
	return map[string]bool{
		"fmt.Print":   true,
		"fmt.Printf":  true,
		"fmt.Println": true,
	}
}

// shouldIgnoreCall проверяет, следует ли игнорировать ошибки от данного вызова.
// Возвращает true, если вызов функции находится в списке игнорируемых функций.
//
// Параметры:
//   - pass: проход анализа
//   - call: выражение вызова для проверки
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
	ignoredFunctions := getIgnoredFunctions()
	for prefix := range ignoredFunctions {
		if strings.HasPrefix(fullName, prefix) {
			return true
		}
	}

	return false
}

// processExprStmt проверяет выражения на необработанные ошибки.
// Сообщает о проблеме, если выражение возвращает ошибку, которая не проверяется.
//
// Параметры:
//   - x: выражение для проверки
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

// processTupleAssign проверяет присваивания кортежей на необработанные ошибки.
// Сообщает о проблеме, если ошибка явно отбрасывается путем присваивания "_".
//
// Параметры:
//   - x: выражение присваивания для проверки
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

// processMultiAssign проверяет множественные присваивания на необработанные ошибки.
// Сообщает о проблеме, если ошибка явно отбрасывается путем присваивания "_".
//
// Параметры:
//   - x: выражение присваивания для проверки
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
// Он направляет обработку в соответствующий обработчик в зависимости от типа узла.
//
// Параметры:
//   - node: узел AST для обработки
//
// Возвращает true для продолжения обхода AST.
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

// run реализует логику анализа для анализатора errcheck.
// Он обходит AST каждого файла в пакете и проверяет наличие необработанных ошибок.
//
// Параметры:
//   - pass: проход анализа, содержащий AST и информацию о типах
//
// Возвращает nil, nil, если проблемы не найдены (стандартное поведение для анализаторов).
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

// resultErrors определяет, какие из возвращаемых значений функции являются ошибками.
// Возвращает массив булевых значений, где true означает, что соответствующее
// возвращаемое значение имеет тип error.
//
// Параметры:
//   - pass: проход анализа, содержащий информацию о типах
//   - call: выражение вызова для проверки
func resultErrors(pass *analysis.Pass, call *ast.CallExpr) []bool {
	// Получаем тип выражения вызова
	callType := pass.TypesInfo.Types[call].Type
	if callType == nil {
		return nil
	}

	// Получаем тип функции
	switch t := callType.(type) {
	case *types.Tuple:
		// Функция возвращает несколько значений
		n := t.Len()
		res := make([]bool, n)
		for i := range n {
			res[i] = isErrorType(t.At(i).Type())
		}
		return res
	default:
		// Функция возвращает одно значение
		return []bool{isErrorType(callType)}
	}
}

// isReturnError проверяет, возвращает ли вызов функции ошибку.
// Возвращает true, если хотя бы одно из возвращаемых значений имеет тип error.
//
// Параметры:
//   - pass: проход анализа, содержащий информацию о типах
//   - call: выражение вызова для проверки
func isReturnError(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Получаем информацию о типах возвращаемых значений
	results := resultErrors(pass, call)
	if results == nil {
		return false
	}

	// Проверяем, есть ли среди возвращаемых значений ошибка
	for _, isErr := range results {
		if isErr {
			return true
		}
	}
	return false
}
