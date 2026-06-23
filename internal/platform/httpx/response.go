package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessReponse struct {
	Data interface{} `json:"data"`
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func WriteSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessReponse{Data: data})
}
