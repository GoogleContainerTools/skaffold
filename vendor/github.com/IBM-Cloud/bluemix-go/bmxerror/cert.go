package bmxerror

//InvalidSSLCert ...
type InvalidSSLCert struct {
	URL    string
	Reason string
}

//NewInvalidSSLCert ...
func NewInvalidSSLCert(url, reason string) *InvalidSSLCert {
	return &InvalidSSLCert{
		URL:    url,
		Reason: reason,
	}
}

func (err *InvalidSSLCert) Error() string {
	message := "Received invalid SSL certificate from " + err.URL
	if err.Reason != "" {
		message += " - " + err.Reason
	}
	return message
}
