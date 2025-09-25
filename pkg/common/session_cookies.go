package common

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
)

func generateSessionId() int {
	return int(time.Now().UnixNano())
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, sessionId int) {
	ca, err := r.Cookie("ca")
	if err != nil {
		return
	}
	if ca.Value != "all" {
		http.SetCookie(w, &http.Cookie{
			Name:     "sid",
			Value:    "",
			Domain:   strings.TrimPrefix(r.Host, "."),
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
			HttpOnly: true,
			MaxAge:   0,
			Path:     "/", //MaxAge: 7200
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    fmt.Sprintf("%d", sessionId),
		Domain:   strings.TrimPrefix(r.Host, "."),
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		HttpOnly: true,
		MaxAge:   2592000000,
		Path:     "/", //MaxAge: 7200
	})
}

func HandleSessionCookie(tracking types.Tracking, w http.ResponseWriter, r *http.Request) int {
	sessionId := generateSessionId()
	c, err := r.Cookie("sid")
	if err != nil {
		// fmt.Printf("Failed to get cookie %v", err)
		if tracking != nil {
			go tracking.TrackSession(sessionId, r)
		}
		setSessionCookie(w, r, sessionId)

	} else {
		sessionId, err = strconv.Atoi(c.Value)
		if err != nil {
			setSessionCookie(w, r, sessionId)
		}
	}
	return sessionId
}
