package node

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
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

func validateURL(urlStr string, validSchemes []string) bool {
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

	for _, scheme := range validSchemes {
		if parsedURL.Scheme == scheme {
			return true
		}
	}
	return false
}

func generateFormatString(schemes []string) string {
	var formatBuilder strings.Builder

	for i, scheme := range schemes {
		if scheme == "" {
			formatBuilder.WriteString("<host>:<port>")
		} else {
			formatBuilder.WriteString(strings.ToUpper(scheme))
		}

		if i < len(schemes)-1 {
			formatBuilder.WriteString(" or ")
		}
	}

	return formatBuilder.String()
}

func RegisterFlagWithValidationOrFail(cmd *cobra.Command, name string, description string, example string, expectedSchemes []string) *string {
	formatExample := generateFormatString(expectedSchemes)
	flagValue := cmd.Flags().String(name, "", fmt.Sprintf("%s.\nFormat: %s. Example: '%s'", description, formatExample, example))

	// Perform validation after flags are parsed
	cobra.OnInitialize(func() {
		if *flagValue == "" || *flagValue == "none" {
			return
		}

		if valid := validateURL(*flagValue, expectedSchemes); !valid {
			log.Fatalf("Invalid format for flag --%s. Expected format: %s. Example: '%s'", name, formatExample, example)
		}
	})

	return flagValue
}
