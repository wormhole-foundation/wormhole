package node

import (
	"fmt"
	"log"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/spf13/cobra"
)

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

		if valid := common.ValidateURL(*flagValue, expectedSchemes); !valid {
			log.Fatalf("Invalid format for flag --%s. Expected format: %s. Example: '%s'", name, formatExample, example)
		}
	})

	return flagValue
}
