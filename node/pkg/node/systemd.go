package node

import (
	"fmt"
	"net"

	"github.com/coreos/go-systemd/activation"
)

func getSDListeners() ([]net.Listener, error) {
	// We use systemd socket activation for (almost) zero downtime deployment -
	// systemd will keep the socket open even while we restart the process
	// (plus, it allows us to bind to unprivileged ports without extra capabilities).
	//
	// Read more: https://vincent.bernat.ch/en/blog/2018-systemd-golang-socket-activation

	listeners, err := activation.Listeners()
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve listeners: %v", err)
	}
	if len(listeners) != 1 {
		return nil, fmt.Errorf("unexpected number of sockets passed by systemd (%d != 1)", len(listeners))
	}

	return listeners, nil
}
