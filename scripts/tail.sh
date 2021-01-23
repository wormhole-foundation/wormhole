#!/usr/bin/env bash

while : ; do
  kubectl logs --tail=1000 --follow=true $1 guardiand
  sleep 1
done
