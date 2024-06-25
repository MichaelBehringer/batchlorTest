package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/response"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/api/api_proxy"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/jwto"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
)

// The internal name of the JWT Cookie
const jwtCookieName = "JWTAuthentication"

// AuthenticationMiddleware is a middleware for validating JWT Tokens.
// Therefore, an "Authorization" header with the "Bearer" schema or a cookie
// with the token is required.
// If no valid token was given, 401 will be returned immediately
func (api *Api) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.Split(r.Header.Get("Authorization"), "Bearer ")
		cookie, errCookie := r.Cookie(jwtCookieName)
		if len(authHeader) != 2 && errCookie != nil {
			if len(authHeader) == 1 {
				response.WriteText("No authorization token or cookie given", 403, w)
			} else {
				logger.Debug("Received malformed JWT token: %s", authHeader)
				response.WriteText("Malformed token", 401, w)
			}

			return
		}
		var token string
		if len(authHeader) == 2 {
			token = authHeader[1]
		} else {
			token = cookie.Value
		}

		claims, authorized, err := jwto.ValidateToken(token, []byte(api.Config.LfsJwtKey))
		if !authorized {
			logger.Debug("Not authorized: %s", err)
			response.WriteText("Unauthorized", 401, w)
		} else {
			user, err := claims.ToUser([]byte(api.Config.LfsJwtKey))
			if err != nil {
				logger.Error("Failed to convert claims to user: %s", err)
				response.WriteText("Unauthorized", 401, w)
			} else {
				// Check if correct db is given for production mode
				if api.Config.Production && user.Database != models.LFS {
					response.WriteText("Invalid db selected in production", 401, w)
					return
				}

				// Set user object accessable for all endpoints
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), models.KeyUser, user)))
			}
		}
	})
}

// Login makes a login request to the LFS service endpoint.
// If the login was successfull, the cookie will be forwarded to the
// own domain
func (api Api) login(w http.ResponseWriter, r *http.Request) {

	// Parse the form
	err := r.ParseForm()
	if err != nil {
		response.WriteText(err.Error(), 400, w)
		return
	}

	// Validate db selection
	if api.Config.Production {
		if strings.ToLower(r.FormValue("db")) != "lfs" {
			response.WriteText("Invalid db selected in production: "+r.FormValue("db"), 400, w)
		}
	}

	// Set JWT version
	r.Form.Set("version", "2")
	body := strings.NewReader(r.Form.Encode())

	// Make the request against the lfs service endpoint
	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, api.Config.LfsServiceEndpoint+"/user/login", body)
	req.Header.Set("Origin", "javalfs")
	if err != nil {
		response.WriteError(err, w, r)
		return
	}
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	res, err := client.Do(req)
	if err != nil {
		logger.Warning("Failed to call login endpoint of LFS: %s", err)
		response.WriteText("Internal Server error", 500, w)
		return
	}
	defer res.Body.Close()

	// Modify the cookie
	if res.StatusCode == 200 {
		cookies := res.Cookies()
		if len(cookies) == 0 {
			logger.Warning("No cookie received from login endpoint on http 200")
			response.WriteText("No cookie set", 500, w)
			return
		}

		cookie := cookies[0]
		logger.Debug("Received cookie %s from login endpoint", cookie.Name)
		cookie.Name = jwtCookieName
		cookie.HttpOnly = true
		cookie.Secure = false
		cookie.SameSite = http.SameSiteStrictMode
		cookie.Domain = ""

		http.SetCookie(w, cookie)
	} else {
		message, _ := io.ReadAll(res.Body)
		logger.Debug("Login failed (received #%d): %s", res.StatusCode, message)
		w.WriteHeader(res.StatusCode)
	}

	// Copy the response
	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	io.Copy(w, res.Body)
}
func (api *Api) queryAuthentication(w http.ResponseWriter, r *http.Request) {
	response.WriteText("Ok", 200, w)
}

func (api *Api) logout(w http.ResponseWriter, r *http.Request) {
	c := &http.Cookie{
		Name:    jwtCookieName,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),

		HttpOnly: true,
	}
	http.SetCookie(w, c)

	// Send logout request to the LFS.X
	r.URL.Path = "/api/host/stop"
	api_proxy.ProxyHost(w, r, api.vncService)

	// Any more calls to the ResponseWriter are not allowed
	//response.WriteText("Cookie deleted", 200, w)
}
