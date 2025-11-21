#!/usr/bin/env bash

# Lint script to check for improper JS package manager usage
# Based on CONTRIBUTING.md supply chain security guidelines
#
# Checks: npm, pnpm, yarn, bun

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
    
    # Skip documentation placeholders
    local skip_placeholders='<package>|<version>'
    
    # NPM violations per CONTRIBUTING.md:
    # - ALLOWED: npm ci (for lockfile installs)
    # - ALLOWED: npm i package@version (pinned package)
    # - ALLOWED: npm i -g package@version (pinned global)
    # - NOT ALLOWED: npm install (bare, should use npm ci)
    # - NOT ALLOWED: npm i package (unpinned package)
    while IFS= read -r line_num; do
        if [[ -n "$line_num" ]]; then
            local line_content
            line_content=$(sed -n "${line_num}p" "$file")
            
            # Skip comments and placeholders
            if echo "$line_content" | grep -qE "^\s*#|$skip_placeholders"; then
                continue
            fi
            
            # Skip if it's npm ci (correct usage)
            if echo "$line_content" | grep -qE 'npm\s+ci'; then
                continue
            fi
            
            # Check if line has npm install or npm i
            if echo "$line_content" | grep -qE 'npm\s+(install|i)(\s|$)'; then
                # Check if it has a version pin (@version)
                if echo "$line_content" | grep -qE '@[0-9]+\.[0-9]+'; then
                    continue
                fi
                
                # Check if it's bare 'npm install' (should use npm ci)
                if echo "$line_content" | grep -qE 'npm\s+(install|i)(\s+)?($|&&|;|\||#)'; then
                    violations+=("$line_num:Use 'npm ci' instead of 'npm install' for lockfile-based installations")
                    continue
                fi
                
                violations+=("$line_num:npm package installation without version pin (use 'npm i package@version')")
            fi
        fi
    done < <(grep -n "\bnpm\s\+\(install\|i\)" "$file" 2>/dev/null | cut -d: -f1)
    
    # PNPM violations:
    # - ALLOWED: pnpm install --frozen-lockfile (respects lockfile)
    # - ALLOWED: pnpm add package@version (pinned package)
    # - ALLOWED: pnpm add -g package@version (pinned global)
    # - NOT ALLOWED: pnpm install (should use --frozen-lockfile)
    # - NOT ALLOWED: pnpm add package (unpinned)
    while IFS= read -r line_num; do
        if [[ -n "$line_num" ]]; then
            local line_content
            line_content=$(sed -n "${line_num}p" "$file")
            
            # Skip comments and placeholders
            if echo "$line_content" | grep -qE "^\s*#|$skip_placeholders"; then
                continue
            fi
            
            # Check for pnpm install without --frozen-lockfile
            if echo "$line_content" | grep -qE 'pnpm\s+install'; then
                if ! echo "$line_content" | grep -qE -- '--frozen-lockfile'; then
                    violations+=("$line_num:Use 'pnpm install --frozen-lockfile' to respect lockfile")
                fi
            fi
            
            # Check for pnpm add without version
            if echo "$line_content" | grep -qE 'pnpm\s+add\s'; then
                if ! echo "$line_content" | grep -qE '@[0-9]+\.[0-9]+'; then
                    violations+=("$line_num:pnpm package installation without version pin (use 'pnpm add package@version')")
                fi
            fi
        fi
    done < <(grep -n "pnpm\s\+\(install\|add\)" "$file" 2>/dev/null | cut -d: -f1)
    
    # YARN violations:
    # - ALLOWED: yarn install --frozen-lockfile or yarn --frozen-lockfile (respects lockfile)
    # - ALLOWED: yarn add package@version (pinned package)
    # - ALLOWED: yarn global add package@version (pinned global)
    # - NOT ALLOWED: yarn install or bare yarn (should use --frozen-lockfile)
    # - NOT ALLOWED: yarn add package (unpinned)
    while IFS= read -r line_num; do
        if [[ -n "$line_num" ]]; then
            local line_content
            line_content=$(sed -n "${line_num}p" "$file")
            
            # Skip comments and placeholders
            if echo "$line_content" | grep -qE "^\s*#|$skip_placeholders"; then
                continue
            fi
            
            # Check for yarn install without --frozen-lockfile or --immutable
            if echo "$line_content" | grep -qE 'yarn(\s+install)?(\s+)?($|&&|;|\||#)'; then
                if ! echo "$line_content" | grep -qE -- '--frozen-lockfile|--immutable'; then
                    violations+=("$line_num:Use 'yarn install --frozen-lockfile' to respect lockfile")
                fi
            fi
            
            # Check for yarn add without version
            if echo "$line_content" | grep -qE 'yarn\s+(global\s+)?add\s'; then
                if ! echo "$line_content" | grep -qE '@[0-9]+\.[0-9]+'; then
                    violations+=("$line_num:yarn package installation without version pin (use 'yarn add package@version')")
                fi
            fi
        fi
    done < <(grep -n "yarn\s*\(\(install\)\|\(add\)\|$\)" "$file" 2>/dev/null | cut -d: -f1)
    
    # BUN violations:
    # - ALLOWED: bun install --frozen-lockfile (respects lockfile)
    # - ALLOWED: bun add package@version (pinned package)
    # - ALLOWED: bun add -g package@version (pinned global)
    # - NOT ALLOWED: bun install (should use --frozen-lockfile)
    # - NOT ALLOWED: bun add package (unpinned)
    while IFS= read -r line_num; do
        if [[ -n "$line_num" ]]; then
            local line_content
            line_content=$(sed -n "${line_num}p" "$file")
            
            # Skip comments and placeholders
            if echo "$line_content" | grep -qE "^\s*#|$skip_placeholders"; then
                continue
            fi
            
            # Check for bun install without --frozen-lockfile
            if echo "$line_content" | grep -qE 'bun\s+install'; then
                if ! echo "$line_content" | grep -qE -- '--frozen-lockfile'; then
                    violations+=("$line_num:Use 'bun install --frozen-lockfile' to respect lockfile")
                fi
            fi
            
            # Check for bun add without version
            if echo "$line_content" | grep -qE 'bun\s+add\s'; then
                if ! echo "$line_content" | grep -qE '@[0-9]+\.[0-9]+'; then
                    violations+=("$line_num:bun package installation without version pin (use 'bun add package@version')")
                fi
            fi
        fi
    done < <(grep -n "bun\s\+\(install\|add\)" "$file" 2>/dev/null | cut -d: -f1)
    
    # Print violations for this file
    if [[ ${#violations[@]} -gt 0 ]]; then
        echo -e "${RED}✗ $file${NC}"
        for violation in "${violations[@]}"; do
            local line_num="${violation%%:*}"
            local message="${violation#*:}"
            local line_content
            line_content=$(sed -n "${line_num}p" "$file" | sed 's/^[[:space:]]*//')
            echo -e "  ${YELLOW}Line $line_num:${NC} $message"
            echo -e "  ${BLUE}>${NC} $line_content"
        done
        echo ""
        VIOLATIONS_FOUND=$((VIOLATIONS_FOUND + ${#violations[@]}))
        return 1
    fi
}

echo -e "${BLUE}Checking for JS package manager violations...${NC}\n"

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
