#!/usr/bin/env bash
set -euo pipefail
# Tilt cannot differentiate between the listen and web address, so we need to jerry-rig the
# external IP onto the external interface and undo the DNAT.

if [[ "$EUID" -eq 0 ]]; then
  echo "Do not run as root"
  exit 1
fi

EXT_IP=$(curl -s -H "Metadata-Flavor: Google" \
  "http://metadata/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip")

IFACE=$(ip route | awk '/default/ { print $5 }')

if [[ -z "${IFACE}" ]]; then
  echo "Could not find interface"
  exit 1
fi

if [[ -z "${EXT_IP}" ]]; then
  echo "Could not find external IP"
  exit 1
fi

if ! ip addr show dev $IFACE | grep -q "inet $EXT_IP"; then
  echo "Adding IP $EXT_IP to $IFACE"
  sudo ip addr add "$EXT_IP/32" dev $IFACE
fi

RULE="-i $IFACE -p tcp ! --dport 22 -j DNAT --to-destination $EXT_IP"
if ! sudo iptables -t nat -C PREROUTING $RULE; then
  echo "Adding iptables rule $RULE"
  sudo iptables -t nat -I PREROUTING $RULE
fi

tilt up --host=$EXT_IP --port=8080 -- "--webHost=$EXT_IP" ${@}
