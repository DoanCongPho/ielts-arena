package httpx

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func IDFromPath(r *http.Request) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
}
