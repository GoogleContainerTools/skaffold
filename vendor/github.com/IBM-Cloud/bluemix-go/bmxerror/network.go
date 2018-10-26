package bmxerror

import (
	"crypto/x509"
	"fmt"
	"net"
	"net/url"

	"golang.org/x/net/websocket"
)

//WrapNetworkErrors ...
func WrapNetworkErrors(host string, err error) error {
	var innerErr error
	switch typedErr := err.(type) {
	case *url.Error:
		innerErr = typedErr.Err
	case *websocket.DialError:
		innerErr = typedErr.Err
	}

	if innerErr != nil {
		switch typedInnerErr := innerErr.(type) {
		case x509.UnknownAuthorityError:
			return NewInvalidSSLCert(host, "unknown authority")
		case x509.HostnameError:
			return NewInvalidSSLCert(host, "not valid for the requested host")
		case x509.CertificateInvalidError:
			return NewInvalidSSLCert(host, "")
		case *net.OpError:
			if typedInnerErr.Op == "dial" {
				return fmt.Errorf("%s\n%s", err.Error(), "TIP: If you are behind a firewall and require an HTTP proxy, verify the https_proxy environment variable is correctly set. Else, check your network connection.")
			}
		}
	}

	return err
}
