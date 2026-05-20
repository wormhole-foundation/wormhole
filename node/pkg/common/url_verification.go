package common

import (
	"errors"
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

// SafeURLForLogging returns only the host and port for a URL-like string.
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

	if host := parsedURL.Host; host != "" {
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

	safeURL := SafeURLForLogging(urlStr)
	errStr := err.Error()

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.URL != "" {
		// url.Error carries the URL separately. Sanitize that exact value in the
		// full error string so wrapped errors keep their additional context.
		errStr = strings.ReplaceAll(errStr, urlErr.URL, SafeURLForLogging(urlErr.URL))
	}

	errStr = strings.ReplaceAll(errStr, urlStr, safeURL)

	parsedURL, parseErr := url.Parse(urlStr)
	if parseErr != nil {
		return errStr
	}

	// Some libraries report URLs through url.URL.String(), which can escape or otherwise
	// canonicalize the original string.
	canonicalURL := parsedURL.String()
	errStr = strings.ReplaceAll(errStr, canonicalURL, safeURL)

	if parsedURL.User != nil {
		if _, hasPassword := parsedURL.User.Password(); hasPassword {
			// net/http redacts userinfo passwords as "***" in *url.Error while leaving
			// path and query values intact, so the raw URL no longer matches exactly.
			redactedURL := strings.Replace(canonicalURL, parsedURL.User.String()+"@", parsedURL.User.Username()+":***@", 1)
			errStr = strings.ReplaceAll(errStr, redactedURL, safeURL)
		}
	}

	return errStr
}
