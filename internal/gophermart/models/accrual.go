package models

import (
	"math"
	"strconv"
)

type AccrualStatus string

const (
	AccrualRegistered AccrualStatus = "REGISTERED"
	AccrualProcessing AccrualStatus = "PROCESSING"
	AccrualInvalid    AccrualStatus = "INVALID"
	AccrualProcessed  AccrualStatus = "PROCESSED"
)

// RoundedFloat64 Определение пользовательского типа для округления
type RoundedFloat64 float64

// MarshalJSON реализует интерфейс json.Marshaler для RoundedFloat64
//
//nolint:unparam // error always nil but required by the json.Marshaler interface
func (f RoundedFloat64) MarshalJSON() ([]byte, error) {
	// Округляем значение до 2 знаков после запятой
	rounded := math.Round(float64(f)*100) / 100

	// Форматируем округленное число в строку
	// 'f' - формат без экспоненты, 2 - точность, 64 - битность float
	s := strconv.FormatFloat(rounded, 'f', 2, 64)

	// Возвращаем строку в виде байтового среза, заключенную в кавычки,
	// чтобы она была валидным JSON-значением (строкой)
	return []byte(s), nil
}

type AccrualResponse struct {
	Number  string         `json:"order"`
	Status  AccrualStatus  `json:"status"`
	Accrual RoundedFloat64 `json:"accrual"`
	Order   OrderProcess   `json:"-"`
}
