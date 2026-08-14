package common

import (
	"net"
	"net/url"
	"strings"
)

func hasKnownSchemePrefix(urlStr string) bool {
	knownSchemes := []string{"http://", "https://", "ws://", "wss://"}
	for _, scheme := range knownSchemes {
		if strings.HasPrefix(urlStr, scheme) {
			return true
		}
	}
	return false
}

func ValidateURL(urlStr string, validSchemes []string) bool {
	if !hasKnownSchemePrefix(urlStr) {
		allowsBare := false
		for _, scheme := range validSchemes {
			if scheme == "" {
				allowsBare = true
				break
			}
		}
		if !allowsBare {
			return false
		}
		host, port, err := net.SplitHostPort(urlStr)
		return err == nil && host != "" && port != ""
	}

	// url.Parse() has to come later because it will fail if the scheme is empty
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	if parsedURL.Host == "" {
		return false
	}

	for _, scheme := range validSchemes {
		if parsedURL.Scheme == scheme {
			return true
		}
	}
	return false
}
