package models

type AccrualStatus string

const (
	AccrualRegistered AccrualStatus = "REGISTERED"
	AccrualProcessing AccrualStatus = "PROCESSING"
	AccrualInvalid    AccrualStatus = "INVALID"
	AccrualProcessed  AccrualStatus = "PROCESSED"
)

type AccrualResponse struct {
	Number  string        `json:"order"`
	Status  AccrualStatus `json:"status"`
	Accrual float64       `json:"accrual"`
	Order   OrderProcess  `json:"-"`
}
