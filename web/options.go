package web

type WebOptions struct {
	Debug bool         `json:"debug"`
	Host  string       `json:"host"`
	Cert  *Certificate `json:"cert"`
}
