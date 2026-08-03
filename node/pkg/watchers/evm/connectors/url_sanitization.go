package connectors

import nodeCommon "github.com/certusone/wormhole/node/pkg/common"

type rpcURLErrorSanitizer interface {
	RPCURL() string
}

func safeConnectorErrorForLogging(err error, connector Connector) string {
	if err == nil {
		return ""
	}

	if sanitizer, ok := connector.(rpcURLErrorSanitizer); ok {
		return nodeCommon.SafeErrorForLogging(err, sanitizer.RPCURL())
	}

	return err.Error()
}
