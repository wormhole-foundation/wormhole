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
	// If no scheme is required, validate host:port format
	if len(validSchemes) == 1 && validSchemes[0] == "" {
		host, port, err := net.SplitHostPort(urlStr)
		return err == nil && host != "" && port != "" && !hasKnownSchemePrefix(urlStr)
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

// SafeURLForLogging returns only the hostname for a URL-like string.
// It intentionally omits userinfo, path, query, and fragment because those may contain credentials.
func SafeURLForLogging(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil || parsedURL.Host == "" {
		// Schemeless host:port strings parse as scheme:opaque. Parse them again
		// as network-path references so URL.Hostname can extract the host.
		parsedURL, err = url.Parse("//" + urlStr)
	}
	if err != nil {
		return "<invalid-url>"
	}

	if host := parsedURL.Hostname(); host != "" {
		return host
	}

	return "<invalid-url>"
}

// SafeErrorForLogging returns an error string with a raw URL replaced by its safe logging form.
func SafeErrorForLogging(err error, urlStr string) string {
	if err == nil {
		return ""
	}
	if urlStr == "" {
		return err.Error()
	}
	return strings.ReplaceAll(err.Error(), urlStr, SafeURLForLogging(urlStr))
}
