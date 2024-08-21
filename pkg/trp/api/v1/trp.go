package api

type TRPVersion struct {
	Version string `json:"version,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
}

type TRPExtensions struct {
	Required  []string `json:"required,omitempty"`
	Supported []string `json:"supported,omitempty"`
}

type Identity struct {
	Name  string `json:"name,omitempty"`
	LEI   string `json:"lei,omitempty"`
	Certs string `json:"x509,omitempty"`
}
