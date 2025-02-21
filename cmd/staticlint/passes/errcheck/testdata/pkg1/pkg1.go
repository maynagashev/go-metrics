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
