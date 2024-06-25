package models

// Login contains informations that are needed for the LFSX
// to perform the login
type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Db       string `json:"db"`
}
