package web

import "os"

type Certificate struct {
	CertFile string `json:"cert"`
	KeyFile  string `json:"key"`

	certFileInfo os.FileInfo
	keyFileInfo  os.FileInfo
}

func NewCertificate(certFile, keyFile string) *Certificate {
	if certFile == "" || keyFile == "" {
		return nil
	}
	return &Certificate{
		CertFile: certFile,
		KeyFile:  keyFile,
	}
}
