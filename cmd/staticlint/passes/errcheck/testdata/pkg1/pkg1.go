package pkg1

import (
	"errors"
	"fmt"
)

func returnsError() error {
	return errors.New("some error")
}

func returnsValueAndError() (string, error) {
	return "", errors.New("some error")
}

func ExampleUncheckedError() {
	// Ожидается ошибка: необработанная ошибка
	returnsError() // want "expression returns unchecked error"

	// Ожидается ошибка: игнорирование ошибки через _
	_ = returnsError() // want "assignment with unchecked error"

	// Ожидается ошибка: игнорирование второго значения (ошибки)
	_, _ = returnsValueAndError() // want "assignment with unchecked error"

	// Правильная обработка ошибок - не должно быть предупреждений
	if err := returnsError(); err != nil {
		fmt.Println(err)
	}

	val, err := returnsValueAndError()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(val)
}

// ExampleMultiAssign тестирует обработку множественных присваиваний
// для проверки функции processMultiAssign
func ExampleMultiAssign() {
	// Множественное присваивание с игнорированием ошибки
	a, _ := returnsValueAndError() // want "assignment with unchecked error"
	fmt.Println(a)

	// Множественное присваивание с правильной обработкой ошибки
	b, err := returnsValueAndError()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(b)

	// Множественное присваивание с несколькими вызовами функций
	c, _ := returnsValueAndError() // want "assignment with unchecked error"
	d, e := "test", returnsError()
	fmt.Println(c, d)
	if e != nil {
		fmt.Println(e)
	}

	// Множественное присваивание с игнорированием ошибки от fmt.Print
	// (должно быть проигнорировано, так как fmt.Print в списке исключений)
	_, _ = fmt.Print(
		"test",
	) // Это должно быть проигнорировано, так как fmt.Print в списке исключений
}
