package common

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"tornberg.me/facet-search/pkg/tracking"
)

func generateSessionId() int {
	return int(time.Now().UnixNano())
}

func setSessionCookie(w http.ResponseWriter, session_id int) {
	http.SetCookie(w, &http.Cookie{
		Name:  "sid",
		Value: fmt.Sprintf("%d", session_id),
		Path:  "/", //MaxAge: 7200
	})
}

func HandleSessionCookie(tracking tracking.Tracking, w http.ResponseWriter, r *http.Request) int {
	session_id := generateSessionId()
	c, err := r.Cookie("sid")
	if err != nil {
		// fmt.Printf("Failed to get cookie %v", err)
		if tracking != nil {
			go tracking.TrackSession(uint32(session_id), r)
		}
		setSessionCookie(w, session_id)

	} else {
		session_id, err = strconv.Atoi(c.Value)
		if err != nil {
			setSessionCookie(w, session_id)
		}
	}
	return session_id
}
