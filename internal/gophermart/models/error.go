package models

type ErrorJSONResponse struct {
	Errors map[string]map[string]string `json:"errors"`
}
