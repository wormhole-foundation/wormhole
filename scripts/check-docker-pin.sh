# This script is checks to that all our Docker images are pinned to a specific SHA256 hash
#
# References as to why...
#   - https://nickjanetakis.com/blog/docker-tip-18-please-pin-your-docker-image-versions
#   - https://snyk.io/blog/10-docker-image-security-best-practices/ (Specifically: USE FIXED TAGS FOR IMMUTABILITY)
#
find ~+ -type f -name "Docker*" -print0 | xargs -0 grep -s "FROM" {} | grep -v scratch | grep -v sha256
if [ $? -eq 0 ]; then
   echo "[!] Unpinned docker files"
   exit 1
else
   echo "[+] No unpinned docker files"
   exit 0
fi