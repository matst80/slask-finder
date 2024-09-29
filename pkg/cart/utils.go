package cart

import (
	"errors"
	"net/http"
	"strconv"
)

func handleCartCookie(idHandler CartIdStorage, w http.ResponseWriter, r *http.Request) (int, error) {
	c, err := r.Cookie("cartid")
	if err != nil {
		if idHandler == nil {
			return 0, errors.New("No id handler")
		}
		cart_id, err := idHandler.GetNextCartId()
		if err != nil {
			return 0, err
		}
		w.Header().Set("Set-Cookie", "cartid="+strconv.Itoa(cart_id)+"; Path=/")
		return cart_id, nil

	}
	cart_id, err := strconv.Atoi(c.Value)
	if err != nil {
		return 0, err
	}
	return cart_id, nil
}
