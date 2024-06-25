package models

import (
	"fmt"
	"strings"

	"gitea.hama.de/LFS/go-logger"
)

// Key used to set this user object to the request context
const KeyUser = "ApiUserContext"

type Database int

const (
	Unknown Database = iota
	PRJ
	MIG
	LFS
)

func NewDatabase(name string) Database {
	switch strings.ToLower(name) {
	case "lfsprj":
		return PRJ
	case "lfsmig":
		return MIG
	case "lfs":
		return LFS
	default:
		logger.Warning("Received invalid database name: %q", name)
		return MIG
	}
}
func (db Database) String() string {
	switch db {
	case PRJ:
		return "PRJ"
	case MIG:
		return "MIG"
	case LFS:
		return "LFS"
	default:
		return "unknown"
	}
}

// User represents a single user who can login to the application
type User struct {
	Username    string   `json:"fullName"`
	Database    Database `json:"-"`
	DatabaseStr string   `json:"db"`
	DbPassword  string   `json:"-"`
	DbUser      string   `json:"user"`
	Workplace   string   `json:"arbeitsplatz"`
	Expiration  int      `json:"expirationTime"`
}

func (u User) String() string {
	return fmt.Sprintf("User:%s\tDatabase:%d\tPassword:%s\tUsername:%s\tUser:%s\t", u.Username, u.Database, u.DbPassword, u.DbUser, u.Workplace)
}

// Identifier returns a string that consists
// out of the users username and the selected database
func (u User) Identifier() string {
	return strings.ToLower(u.DbUser + "-" + u.Database.String())
}
