package api

import (
	"encoding/json"
	"net/http"
)

func jsonDecode(r *http.Request, out any) error {
	return json.NewDecoder(r.Body).Decode(out)
}
