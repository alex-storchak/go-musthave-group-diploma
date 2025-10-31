package models

type AccrualStatus string

const (
	AccrualRegistered AccrualStatus = "REGISTERED"
	AccrualProcessing               = "PROCESSING"
	AccrualInvalid                  = "INVALID"
	AccrualProcessed                = "PROCESSED"
)

type AccrualResponse struct {
	Number  string        `json:"order"`
	Status  AccrualStatus `json:"status"`
	Accrual float64       `json:"accrual"`
	Order   OrderProcess  `json:"-"`
}
