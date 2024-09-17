package tools

import (
	"strconv"

	"github.com/joeljunstrom/go-luhn"
)

func MyLuhnCheck(number string) bool {
	sum := 0
	shouldDouble := false

	// Прохожусь по строке с конца
	for i := len(number) - 1; i >= 0; i-- {
		// Преобразую символ в число
		digit, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return false
		}

		if shouldDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		shouldDouble = !shouldDouble
	}

	// Проверяю делимость суммы на 10
	return sum%10 == 0
}

// LuhnCheck проверяет номер с использованием алгоритма Луна
func LuhnCheck(number string) bool {
	// использую готовое решение "github.com/joeljunstrom/go-luhn"
	return luhn.Valid(number)
}
