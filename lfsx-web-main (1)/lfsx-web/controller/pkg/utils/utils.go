package utils

// @TODO refactor env to work with struct tags and source "generic" utils out

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/errors"
)

// GetEnvString tries to get an environment variable from the system
// as a string value. If the env was not found the given default value
// will be returned
func GetEnvString(name string, defaultValue string) string {
	val := defaultValue
	if strVal, isSet := os.LookupEnv(name); isSet {
		val = strVal
	}

	return val
}

// RequireEnvString returns the environment variable with the given name.
// If it could not be found, a fatal error will be logged and the program stops
func RequireEnvString(name string) string {
	if strVal, isSet := os.LookupEnv(name); isSet {
		return strVal
	} else {
		logger.Fatal("Required environment variable %q not set", name)
		return ""
	}
}

// GetEnvBool tries to get an environment variable from the system
// as a boolean value. If the env was not found the given default value
// will be returned
func GetEnvBool(name string, defaultValue bool) bool {
	val := defaultValue
	if strVal, isSet := os.LookupEnv(name); isSet {
		strVal = strings.ToLower(strVal)
		return strVal == "1" || strVal == "true" || strVal == "yes" || strVal == "ja"
	}

	return val
}

// GetEnvInt tries to get an environment variable from the system
// as a POSITIVE integer. If the env was not found or is an invalid number,
// the given default value will be returned
func GetEnvInt(name string, defaultValue int) int {
	val := defaultValue
	if strVal, isSet := os.LookupEnv(name); isSet {
		if intVal, err := strconv.Atoi(strVal); err != nil {
			logger.Error("Invalid number value given for the environment variable 'COPY_TABLE_MAX_INSERT_COUNT': %s", strVal)
		} else if val < 1 {
			logger.Error("Environment variable %q has to be greater than 0", name)
		} else {
			val = intVal
		}
	}

	return val
}

// DecodeBody Parses the given Body to the struct
func DecodeBody[T any](str *T, req *http.Request) (*T, error) {
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(str)
	if err != nil {
		logger.Debug("Unable to parse body: %s", err)
		return nil, errors.BadRequest(err.Error())
	}

	return str, nil
}

// GenerateRandomString returns a securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789abcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}

// GetQueryValueInt returns a query value as a number.
// It panics if no integer was provided.
// If the query value was not specified, the default value will be used
func GetQueryValueInt(key string, def int, r *http.Request) int {
	strVal := r.URL.Query().Get(key)

	if strVal == "" || strVal == "undefined" || strVal == "null" {
		return def
	} else {
		rtc, err := strconv.Atoi(strVal)
		if err != nil {
			logger.Debug("Invalid number provided for query value %q: %q", key, strVal)
			panic(errors.BadRequest(fmt.Sprintf("Invalid number provided for query value %q: %q", key, strVal)))
		} else {
			return rtc
		}
	}
}
