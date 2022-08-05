package discord

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type DiscordNotifier struct {
	c      *api.Client
	chans  []discord.Channel
	logger *zap.Logger

	groupToIDMu sync.RWMutex
	groupToID   map[string]string
}

// NewDiscordNotifier returns and initializes a new Discord notifier.
//
// During initialization, a list of all guilds and channels is fetched.
// Newly added guilds and channels won't be detected at runtime.
func NewDiscordNotifier(botToken string, channelName string, logger *zap.Logger) (*DiscordNotifier, error) {
	c := api.NewClient("Bot " + botToken)
	chans := make([]discord.Channel, 0)

	guilds, err := c.Guilds(0)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve guilds: %w", err)
	}

	for _, guild := range guilds {
		gcn, err := c.Channels(guild.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve channels for %s: %w", guild.ID, err)
		}

		for _, cn := range gcn {
			if cn.Name == channelName {
				chans = append(chans, cn)
			}
		}
	}

	logger.Info("notification channels", zap.Any("channels", chans))

	return &DiscordNotifier{
		c:         c,
		chans:     chans,
		logger:    logger,
		groupToID: make(map[string]string),
	}, nil
}

func wrapCode(in string) string {
	return fmt.Sprintf("`%s`", in)
}

func (d *DiscordNotifier) LookupGroupID(groupName string) (string, error) {
	d.groupToIDMu.RLock()
	if id, ok := d.groupToID[groupName]; ok {
		d.groupToIDMu.RUnlock()
		return id, nil
	}
	d.groupToIDMu.RUnlock()

	guilds, err := d.c.Guilds(0)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve guilds: %w", err)
	}

	for _, guild := range guilds {
		gcn, err := d.c.Roles(guild.ID)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve roles for %s: %w", guild.ID, err)
		}

		for _, cn := range gcn {
			if cn.Name == groupName {
				m := cn.ID.String()

				d.groupToIDMu.Lock()
				d.groupToID[groupName] = m
				d.groupToIDMu.Unlock()

				return m, nil
			}
		}
	}

	return "", fmt.Errorf("failed to find group %s", groupName)
}

// Observation defines the same interface as processor.Observation but redefined
// here to avoid circular dependencies.
type Observation interface {
	GetEmitterChain() vaa.ChainID
	MessageID() string
	SigningMsg() common.Hash
}

func (d *DiscordNotifier) MissingSignaturesOnObservation(o Observation, hasSigs, wantSigs int, quorum bool, missing []string) error {
	if len(missing) == 0 {
		panic("no missing nodes specified")
	}
	var quorumText string
	if quorum {
		quorumText = fmt.Sprintf("✔️ yes (%d/%d)", hasSigs, wantSigs)
	} else {
		quorumText = fmt.Sprintf("🚨️ **NO** (%d/%d)", hasSigs, wantSigs)
	}

	var messageText string
	if !quorum {
		messageText = "**NO QUORUM** - Wormhole likely failed to achieve consensus on this message @here"
	}

	missingText := &bytes.Buffer{}
	for _, m := range missing {
		groupID, err := d.LookupGroupID(m)
		if err != nil {
			d.logger.Error("failed to lookup group id", zap.Error(err), zap.String("name", m))
			groupID = m
		} else {
			groupID = fmt.Sprintf("<@&%s>", groupID)
		}

		if _, err := fmt.Fprintf(missingText, "- %s\n", groupID); err != nil {
			panic(err)
		}
	}

	for _, cn := range d.chans {
		if _, err := d.c.SendMessage(cn.ID, messageText,
			discord.Embed{
				Title: "Message with missing signatures",
				Fields: []discord.EmbedField{
					{Name: "Message ID", Value: wrapCode(o.MessageID()), Inline: true},
					{Name: "Digest", Value: wrapCode(hex.EncodeToString(o.SigningMsg().Bytes())), Inline: true},
					{Name: "Quorum", Value: quorumText, Inline: true},
					{Name: "Source Chain", Value: strings.Title(o.GetEmitterChain().String()), Inline: false},
					{Name: "Missing Guardians", Value: missingText.String(), Inline: false},
				},
			},
		); err != nil {
			return err
		}
	}

	return nil
}
