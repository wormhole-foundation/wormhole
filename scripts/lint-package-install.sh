#!/usr/bin/env bash

# Lint script to check for improper npm usage
# Based on CONTRIBUTING.md supply chain security guidelines

set -euo pipefail

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
VIOLATIONS_FOUND=0
FILES_CHECKED=0

# Function to check a file for violations
check_file() {
    local file="$1"
    local violations=()
    
    FILES_CHECKED=$((FILES_CHECKED + 1))
    
    # Skip if file doesn't exist or is binary
    [[ ! -f "$file" ]] && return
    file "$file" | grep -q "text" || return
    
    # Check for npm violations per CONTRIBUTING.md:
    # - ALLOWED: npm ci (for lockfile installs)
    # - ALLOWED: npm i package@version or npm install package@version (pinned package)
    # - ALLOWED: npm i -g package@version or npm install -g package@version (pinned global)
    # - NOT ALLOWED: npm install (bare, should use npm ci)
    # - NOT ALLOWED: npm i package or npm install package (unpinned package)
    while IFS= read -r line_num; do
        if [[ -n "$line_num" ]]; then
            local line_content=$(sed -n "${line_num}p" "$file")
            
            # Skip comments
            if echo "$line_content" | grep -qE '^\s*#'; then
                continue
            fi
            
            # Skip if it's npm ci (correct usage)
            if echo "$line_content" | grep -qE 'npm\s+ci'; then
                continue
            fi
            
            # Check if line has npm install or npm i (with or without trailing space)
            if echo "$line_content" | grep -qE 'npm\s+(install|i)(\s|$)'; then
                # Skip documentation placeholders like <package>, <version>
                if echo "$line_content" | grep -qE '<package>|<version>'; then
                    continue
                fi
                
                # Check if it has a version pin (@version)
                if echo "$line_content" | grep -qE '@[0-9]+\.[0-9]+'; then
                    # Has version pin - this is OK
                    continue
                fi
                
                # Check if it's bare 'npm install' or 'npm i' with no package name
                # This should use npm ci instead
                if echo "$line_content" | grep -qE 'npm\s+(install|i)(\s+)?($|&&|;|\||#)'; then
                    violations+=("$line_num:Use 'npm ci' instead of 'npm install' for lockfile-based installations")
                    continue
                fi
                
                # If we get here, it's npm install/i with a package name but no @version
                violations+=("$line_num:npm package installation without version pin (use 'npm i package@version' to pin version)")
            fi
        fi
    done < <(grep -n "npm\s\+\(install\|i\)" "$file" 2>/dev/null | cut -d: -f1)
    
    # Print violations for this file
    if [[ ${#violations[@]} -gt 0 ]]; then
        echo -e "${RED}✗ $file${NC}"
        for violation in "${violations[@]}"; do
            local line_num="${violation%%:*}"
            local message="${violation#*:}"
            local line_content=$(sed -n "${line_num}p" "$file" | sed 's/^[[:space:]]*//')
            echo -e "  ${YELLOW}Line $line_num:${NC} $message"
            echo -e "  ${BLUE}>${NC} $line_content"
        done
        echo ""
        VIOLATIONS_FOUND=$((VIOLATIONS_FOUND + ${#violations[@]}))
        return 1
    fi
}

echo -e "${BLUE}Checking for npm usage violations...${NC}\n"

# Find and check files (exclude node_modules, .git, and this script)
while IFS= read -r file; do
    check_file "$file" || true
done < <(find . -type f \
    -not -path "*/node_modules/*" \
    -not -path "*/.git/*" \
    -not -name "lint-package-install.sh" \
    \( \
        -name "Dockerfile*" -o \
        -name "*.dockerfile" -o \
        -name "*.md" -o \
        -name "*.sh" -o \
        \( \( -name "*.yml" -o -name "*.yaml" \) -path "*/.github/workflows/*" \) \
    \) 2>/dev/null)

# Print summary
echo ""
echo -e "${BLUE}═══════════════════════════════════════${NC}"
if [[ $VIOLATIONS_FOUND -eq 0 ]]; then
    echo -e "${GREEN}✓ No violations found!${NC}"
    echo -e "${BLUE}Files checked: $FILES_CHECKED${NC}"
    exit 0
else
    echo -e "${RED}✗ Found $VIOLATIONS_FOUND violation(s) in $FILES_CHECKED files${NC}"
    echo ""
    echo -e "${YELLOW}Please review CONTRIBUTING.md for supply chain security guidelines${NC}"
    echo ""
    exit 1
fi
