package ccq

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"github.com/gagliardetto/solana-go"
	"gopkg.in/godo.v2/watcher/fswatch"
)

type (
	Config struct {
		Permissions []User `json:"Permissions"`
	}

	User struct {
		UserName      string        `json:"userName"`
		ApiKey        string        `json:"apiKey"`
		AllowUnsigned bool          `json:"allowUnsigned"`
		AllowAnything bool          `json:"allowAnything"`
		LogResponses  bool          `json:"logResponses"`
		AllowedCalls  []AllowedCall `json:"allowedCalls"`
	}

	AllowedCall struct {
		EthCall             *EthCall             `json:"ethCall"`
		EthCallByTimestamp  *EthCallByTimestamp  `json:"ethCallByTimestamp"`
		EthCallWithFinality *EthCallWithFinality `json:"ethCallWithFinality"`
		SolanaAccount       *SolanaAccount       `json:"solAccount"`
		SolanaPda           *SolanaPda           `json:"solPDA"`
	}

	EthCall struct {
		Chain           int    `json:"chain"`
		ContractAddress string `json:"contractAddress"`
		Call            string `json:"call"`
	}

	EthCallByTimestamp struct {
		Chain           int    `json:"chain"`
		ContractAddress string `json:"contractAddress"`
		Call            string `json:"call"`
	}

	EthCallWithFinality struct {
		Chain           int    `json:"chain"`
		ContractAddress string `json:"contractAddress"`
		Call            string `json:"call"`
	}

	SolanaAccount struct {
		Chain   int    `json:"chain"`
		Account string `json:"account"`
	}

	SolanaPda struct {
		Chain          int    `json:"chain"`
		ProgramAddress string `json:"programAddress"`
		// As a future enhancement, we may want to specify the allowed seeds.
	}

	PermissionsMap map[string]*permissionEntry

	permissionEntry struct {
		userName      string
		apiKey        string
		allowUnsigned bool
		allowAnything bool
		logResponses  bool
		allowedCalls  allowedCallsForUser // Key is something like "ethCall:2:000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d6:06fdde03"
	}

	allowedCallsForUser map[string]struct{}

	Permissions struct {
		lock          sync.Mutex
		permMap       PermissionsMap
		fileName      string
		allowAnything bool
		watcher       *fswatch.Watcher
	}
)

// NewPermissions creates a Permissions object which contains the per-user permissions.
func NewPermissions(fileName string, allowAnything bool) (*Permissions, error) {
	permMap, err := parseConfigFile(fileName, allowAnything)
	if err != nil {
		return nil, err
	}

	return &Permissions{
		permMap:       permMap,
		fileName:      fileName,
		allowAnything: allowAnything,
	}, nil
}

// StartWatcher creates an fswatcher to watch for updates to the permissions file and reload it when it changes.
func (perms *Permissions) StartWatcher(ctx context.Context, logger *zap.Logger, errC chan error) {
	logger = logger.With(zap.String("component", "perms"))
	perms.watcher = fswatch.NewWatcher(perms.fileName)
	fsChan := perms.watcher.Start()

	common.RunWithScissors(ctx, errC, "perm_file_watcher", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case notif := <-fsChan:
				if notif.Path != perms.fileName {
					return fmt.Errorf("permissions watcher received an update for an unexpected file: %s", notif.Path)
				}

				logger.Info("the permissions file has been updated", zap.String("fileName", notif.Path), zap.Int("event", int(notif.Event)))
				perms.Reload(logger)
			}
		}
	})
}

// Reload reloads the permissions file.
func (perms *Permissions) Reload(logger *zap.Logger) {
	permMap, err := parseConfigFile(perms.fileName, perms.allowAnything)
	if err != nil {
		logger.Error("failed to reload the permissions file, sticking with the old one", zap.String("fileName", perms.fileName), zap.Error(err))
		permissionFileReloadsFailure.Inc()
		return
	}

	logger.Info("successfully reloaded the permissions file, switching to it", zap.String("fileName", perms.fileName))
	perms.lock.Lock()
	perms.permMap = permMap
	perms.lock.Unlock()
	permissionFileReloadsSuccess.Inc()
}

// StopWatcher stops the permissions file watcher.
func (perms *Permissions) StopWatcher() {
	if perms.watcher != nil {
		perms.watcher.Stop()
	}
}

// GetUserEntry returns the permissions entry for a given API key. It uses the lock to protect against updates.
func (perms *Permissions) GetUserEntry(apiKey string) (*permissionEntry, bool) {
	perms.lock.Lock()
	defer perms.lock.Unlock()
	userEntry, exists := perms.permMap[apiKey]
	return userEntry, exists
}

const ETH_CALL_SIG_LENGTH = 4

// parseConfigFile parses the permissions config file into a map keyed by API key.
func parseConfigFile(fileName string, allowAnything bool) (PermissionsMap, error) {
	jsonFile, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf(`failed to open permissions file "%s": %w`, fileName, err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf(`failed to read permissions file "%s": %w`, fileName, err)
	}

	retVal, err := parseConfig(byteValue, allowAnything)
	if err != nil {
		return retVal, fmt.Errorf(`failed to parse permissions file "%s": %w`, fileName, err)
	}

	return retVal, err
}

// parseConfig parses the permissions config from a buffer into a map keyed by API key.
func parseConfig(byteValue []byte, allowAnything bool) (PermissionsMap, error) {
	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		return nil, fmt.Errorf(`failed to unmarshal json: %w`, err)
	}

	ret := make(PermissionsMap)
	userNames := map[string]struct{}{}
	for _, user := range config.Permissions {
		// Since we log user names in all our error messages, make sure they are unique.
		if _, exists := userNames[user.UserName]; exists {
			return nil, fmt.Errorf(`UserName "%s" is a duplicate`, user.UserName)
		}
		userNames[user.UserName] = struct{}{}

		apiKey := strings.ToLower(user.ApiKey)
		if _, exists := ret[apiKey]; exists {
			return nil, fmt.Errorf(`API key "%s" is a duplicate`, apiKey)
		}

		if user.AllowAnything {
			if !allowAnything {
				return nil, fmt.Errorf(`UserName "%s" has "allowAnything" specified when the feature is not enabled`, user.UserName)
			}
			if len(user.AllowedCalls) != 0 {
				return nil, fmt.Errorf(`UserName "%s" has "allowedCalls" specified with "allowAnything", which is not allowed`, user.UserName)
			}
		}

		// Build the list of allowed calls for this API key.
		allowedCalls := make(allowedCallsForUser)
		for _, ac := range user.AllowedCalls {
			var chain int
			var callType, contractAddressStr, callStr, callKey string
			// var contractAddressStr string
			if ac.EthCall != nil {
				callType = "ethCall"
				chain = ac.EthCall.Chain
				contractAddressStr = ac.EthCall.ContractAddress
				callStr = ac.EthCall.Call
			} else if ac.EthCallByTimestamp != nil {
				callType = "ethCallByTimestamp"
				chain = ac.EthCallByTimestamp.Chain
				contractAddressStr = ac.EthCallByTimestamp.ContractAddress
				callStr = ac.EthCallByTimestamp.Call
			} else if ac.EthCallWithFinality != nil {
				callType = "ethCallWithFinality"
				chain = ac.EthCallWithFinality.Chain
				contractAddressStr = ac.EthCallWithFinality.ContractAddress
				callStr = ac.EthCallWithFinality.Call
			} else if ac.SolanaAccount != nil {
				// We assume the account is base58, but if it starts with "0x" it should be 32 bytes of hex.
				account := ac.SolanaAccount.Account
				if strings.HasPrefix(account, "0x") {
					buf, err := hex.DecodeString(account[2:])
					if err != nil {
						return nil, fmt.Errorf(`invalid solana account hex string "%s" for user "%s": %w`, account, user.UserName, err)
					}
					if len(buf) != query.SolanaPublicKeyLength {
						return nil, fmt.Errorf(`invalid solana account hex string "%s" for user "%s, must be %d bytes`, account, user.UserName, query.SolanaPublicKeyLength)
					}
					account = solana.PublicKey(buf).String()
				} else {
					// Make sure it is valid base58.
					_, err := solana.PublicKeyFromBase58(account)
					if err != nil {
						return nil, fmt.Errorf(`solana account string "%s" for user "%s" is not valid base58: %w`, account, user.UserName, err)
					}
				}
				callKey = fmt.Sprintf("solAccount:%d:%s", ac.SolanaAccount.Chain, account)
			} else if ac.SolanaPda != nil {
				// We assume the account is base58, but if it starts with "0x" it should be 32 bytes of hex.
				pa := ac.SolanaPda.ProgramAddress
				if strings.HasPrefix(pa, "0x") {
					buf, err := hex.DecodeString(pa[2:])
					if err != nil {
						return nil, fmt.Errorf(`invalid solana program address hex string "%s" for user "%s": %w`, pa, user.UserName, err)
					}
					if len(buf) != query.SolanaPublicKeyLength {
						return nil, fmt.Errorf(`invalid solana program address hex string "%s" for user "%s, must be %d bytes`, pa, user.UserName, query.SolanaPublicKeyLength)
					}
					pa = solana.PublicKey(buf).String()
				} else {
					// Make sure it is valid base58.
					_, err := solana.PublicKeyFromBase58(pa)
					if err != nil {
						return nil, fmt.Errorf(`solana program address string "%s" for user "%s" is not valid base58: %w`, pa, user.UserName, err)
					}
				}
				callKey = fmt.Sprintf("solPDA:%d:%s", ac.SolanaPda.Chain, pa)
			} else {
				return nil, fmt.Errorf(`unsupported call type for user "%s", must be "ethCall", "ethCallByTimestamp", "ethCallWithFinality", "solAccount" or "solPDA"`, user.UserName)
			}

			if callKey == "" {
				// Convert the contract address into a standard format like "000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d6".
				contractAddress, err := vaa.StringToAddress(contractAddressStr)
				if err != nil {
					return nil, fmt.Errorf(`invalid contract address "%s" for user "%s"`, contractAddressStr, user.UserName)
				}

				// The call should be the ABI four byte hex hash of the function signature. Parse it into a standard form of "06fdde03".
				call, err := hex.DecodeString(strings.TrimPrefix(callStr, "0x"))
				if err != nil {
					return nil, fmt.Errorf(`invalid eth call "%s" for user "%s"`, callStr, user.UserName)
				}
				if len(call) != ETH_CALL_SIG_LENGTH {
					return nil, fmt.Errorf(`eth call "%s" for user "%s" has an invalid length, must be %d bytes`, callStr, user.UserName, ETH_CALL_SIG_LENGTH)
				}

				// The permission key is the chain, contract address and call formatted as a colon separated string.
				callKey = fmt.Sprintf("%s:%d:%s:%s", callType, chain, contractAddress, hex.EncodeToString(call))
			}

			if _, exists := allowedCalls[callKey]; exists {
				return nil, fmt.Errorf(`"%s" is a duplicate allowed call for user "%s"`, callKey, user.UserName)
			}

			allowedCalls[callKey] = struct{}{}
		}

		pe := &permissionEntry{
			userName:      user.UserName,
			apiKey:        apiKey,
			allowUnsigned: user.AllowUnsigned,
			allowAnything: user.AllowAnything,
			logResponses:  user.LogResponses,
			allowedCalls:  allowedCalls,
		}

		ret[apiKey] = pe
	}

	return ret, nil
}
