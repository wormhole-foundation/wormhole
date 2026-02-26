use std::env;
use std::fs;
use std::path::PathBuf;

/// Represents a chain definition parsed from the Go source
#[derive(Debug, Clone)]
struct ChainDef {
    /// Variant name for the Rust enum (e.g., "Solana")
    name: String,
    /// The chain ID number
    id: u16,
    /// Status of the chain
    status: ChainStatus,
    /// Optional comment/warning from Go source
    comment: Option<String>,
}

#[derive(Debug, Clone, PartialEq)]
enum ChainStatus {
    /// Active chain
    Active,
    /// Marked as OBSOLETE
    Obsolete,
    /// Only supported in devnet/Tilt
    DevnetOnly,
    /// Chain ID reserved but never deployed
    NeverDeployed,
}

impl ChainDef {
    /// Extract the chain name from Go constant name
    /// "ChainIDSolana" -> "Solana"
    fn extract_name(const_name: &str) -> String {
        const_name
            .strip_prefix("ChainID")
            .unwrap_or(const_name)
            .to_string()
    }

    /// Convert name to lowercase for FromStr matching
    fn name_lower(&self) -> String {
        self.name.to_lowercase()
    }

    /// Check if name needs special casing (like "BSC", "TON", "BOB")
    fn needs_special_case(&self) -> bool {
        matches!(
            self.name.as_str(),
            "BSC" | "TON" | "BOB" | "XRPLEVM" | "SeiEVM" | "HyperEVM" | "XLayer"
        )
    }
}

/// Parse the Go source file and extract chain definitions
fn parse_go_constants(go_source: &str) -> Vec<ChainDef> {
    let mut chains = Vec::new();
    let mut pending_comment: Option<(ChainStatus, String)> = None;

    for line in go_source.lines() {
        let trimmed = line.trim();

        // Skip empty lines and non-relevant comments
        if trimmed.is_empty() {
            continue;
        }

        // Skip ChainIDUnset - we use Chain::Any for ID 0
        if trimmed.contains("ChainIDUnset") {
            continue;
        }

        // Check for OBSOLETE comment
        if let Some(caps) = trimmed.strip_prefix("// OBSOLETE:") {
            if let Some((name, id)) = parse_obsolete_line(caps) {
                chains.push(ChainDef {
                    name,
                    id,
                    status: ChainStatus::Obsolete,
                    comment: Some(format!("OBSOLETE: was ID {}", id)),
                });
            }
            continue;
        }

        // Check for WARNING comment (devnet only)
        if let Some(caps) = trimmed.strip_prefix("// WARNING:") {
            pending_comment = Some((ChainStatus::DevnetOnly, caps.trim().to_string()));
            continue;
        }

        // Check for NOTE comment (never deployed)
        if let Some(caps) = trimmed.strip_prefix("// NOTE:") {
            if caps.contains("never deployed") {
                pending_comment = Some((ChainStatus::NeverDeployed, caps.trim().to_string()));
            } else {
                // Regular NOTE comment, clear any pending comment
                pending_comment = None;
            }
            continue;
        }

        // Check for regular comment (chain description) - these should clear pending state
        if trimmed.starts_with("//") && !trimmed.starts_with("// OBSOLETE") {
            // This is a descriptive comment, not a status marker
            // Clear any pending status (prevents NOTE/WARNING from bleeding into next chain)
            pending_comment = None;
            continue;
        }

        // Parse active chain constant: ChainIDSolana ChainID = 1
        if trimmed.starts_with("ChainID") && trimmed.contains(" ChainID = ") {
            if let Some((name, id)) = parse_chain_constant(trimmed) {
                let (status, comment) = if let Some((s, c)) = pending_comment.take() {
                    (s, Some(c))
                } else {
                    (ChainStatus::Active, None)
                };

                chains.push(ChainDef {
                    name,
                    id,
                    status,
                    comment,
                });
            }
            continue;
        }

        // Reset pending comment if we hit something else
        if !trimmed.starts_with("//") {
            pending_comment = None;
        }
    }

    chains
}

/// Parse an obsolete chain line: "// OBSOLETE: ChainIDOasis ChainID = 7"
fn parse_obsolete_line(line: &str) -> Option<(String, u16)> {
    let parts: Vec<&str> = line.split_whitespace().collect();

    // Find "ChainID" followed by "ChainID" and "=" and a number
    for i in 0..parts.len() {
        if parts[i].starts_with("ChainID")
            && i + 3 < parts.len()
            && parts[i + 1] == "ChainID"
            && parts[i + 2] == "="
        {
            let const_name = parts[i];
            let id_str = parts[i + 3].trim_end_matches(|c| !char::is_numeric(c));
            if let Ok(id) = id_str.parse::<u16>() {
                return Some((ChainDef::extract_name(const_name), id));
            }
        }
    }
    None
}

/// Parse a chain constant line: "ChainIDSolana ChainID = 1"
fn parse_chain_constant(line: &str) -> Option<(String, u16)> {
    let parts: Vec<&str> = line.split_whitespace().collect();

    if parts.len() >= 4 && parts[1] == "ChainID" && parts[2] == "=" {
        let const_name = parts[0];
        let id_str = parts[3].trim_end_matches(|c| !char::is_numeric(c));
        if let Ok(id) = id_str.parse::<u16>() {
            return Some((ChainDef::extract_name(const_name), id));
        }
    }
    None
}

/// Generate the complete Rust source code
fn generate_rust_code(chains: &[ChainDef]) -> String {
    let mut code = String::new();

    // Header comment
    code.push_str("// This file is AUTO-GENERATED by build.rs\n");
    code.push_str("// Source: sdk/vaa/structs.go\n");
    code.push_str("// DO NOT EDIT MANUALLY\n\n");

    code.push_str(&generate_enum(chains));
    code.push_str("\n\n");
    code.push_str(&generate_from_u16(chains));
    code.push_str("\n\n");
    code.push_str(&generate_into_u16(chains));
    code.push_str("\n\n");
    code.push_str(&generate_display(chains));
    code.push_str("\n\n");
    code.push_str(&generate_from_str(chains));

    code
}

/// Generate the Chain enum definition
fn generate_enum(chains: &[ChainDef]) -> String {
    let mut code = String::new();

    code.push_str("#[derive(Debug, Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]\n");
    code.push_str("pub enum Chain {\n");
    code.push_str("    /// In the wormhole wire format, 0 indicates that a message is for any destination chain\n");
    code.push_str("    #[default]\n");
    code.push_str("    Any,\n\n");

    // Group chains and add comments for obsolete ones
    let active_chains: Vec<_> = chains
        .iter()
        .filter(|c| c.status == ChainStatus::Active || c.status == ChainStatus::DevnetOnly)
        .collect();

    let obsolete_chains: Vec<_> = chains
        .iter()
        .filter(|c| c.status == ChainStatus::Obsolete)
        .collect();

    // Add active chains with comments
    for chain in active_chains {
        if let Some(comment) = &chain.comment {
            code.push_str(&format!("    /// {}\n", comment));
        }
        code.push_str(&format!("    {},\n", chain.name));
    }

    // Add obsolete chains as comments
    if !obsolete_chains.is_empty() {
        code.push_str("\n    // Obsolete chains:\n");
        for chain in obsolete_chains {
            code.push_str(&format!(
                "    // OBSOLETE: {} was ID {}\n",
                chain.name, chain.id
            ));
        }
    }

    code.push_str("\n    // Allow arbitrary u16s to support future chains\n");
    code.push_str("    Unknown(u16),\n");
    code.push_str("}\n");

    code
}

/// Generate From<u16> for Chain implementation
fn generate_from_u16(chains: &[ChainDef]) -> String {
    let mut code = String::new();

    code.push_str("impl From<u16> for Chain {\n");
    code.push_str("    fn from(other: u16) -> Chain {\n");
    code.push_str("        match other {\n");
    code.push_str("            0 => Chain::Any,\n");

    // Only include active chains in the match
    let active_chains: Vec<_> = chains
        .iter()
        .filter(|c| c.status == ChainStatus::Active || c.status == ChainStatus::DevnetOnly)
        .collect();

    for chain in active_chains {
        code.push_str(&format!(
            "            {} => Chain::{},\n",
            chain.id, chain.name
        ));
    }

    code.push_str("            c => Chain::Unknown(c),\n");
    code.push_str("        }\n");
    code.push_str("    }\n");
    code.push_str("}\n");

    code
}

/// Generate From<Chain> for u16 implementation
fn generate_into_u16(chains: &[ChainDef]) -> String {
    let mut code = String::new();

    code.push_str("impl From<Chain> for u16 {\n");
    code.push_str("    fn from(other: Chain) -> u16 {\n");
    code.push_str("        match other {\n");
    code.push_str("            Chain::Any => 0,\n");

    let active_chains: Vec<_> = chains
        .iter()
        .filter(|c| c.status == ChainStatus::Active || c.status == ChainStatus::DevnetOnly)
        .collect();

    for chain in active_chains {
        code.push_str(&format!(
            "            Chain::{} => {},\n",
            chain.name, chain.id
        ));
    }

    code.push_str("            Chain::Unknown(c) => c,\n");
    code.push_str("        }\n");
    code.push_str("    }\n");
    code.push_str("}\n");

    code
}

/// Generate Display trait implementation
fn generate_display(chains: &[ChainDef]) -> String {
    let mut code = String::new();

    code.push_str("impl fmt::Display for Chain {\n");
    code.push_str("    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {\n");
    code.push_str("        match self {\n");
    code.push_str("            Self::Any => f.write_str(\"Any\"),\n");

    let active_chains: Vec<_> = chains
        .iter()
        .filter(|c| c.status == ChainStatus::Active || c.status == ChainStatus::DevnetOnly)
        .collect();

    for chain in active_chains {
        code.push_str(&format!(
            "            Self::{} => f.write_str(\"{}\"),\n",
            chain.name, chain.name
        ));
    }

    code.push_str("            Self::Unknown(v) => write!(f, \"Unknown({})\", v),\n");
    code.push_str("        }\n");
    code.push_str("    }\n");
    code.push_str("}\n");

    code
}

/// Generate FromStr trait implementation with case-insensitive matching
fn generate_from_str(chains: &[ChainDef]) -> String {
    let mut code = String::new();

    code.push_str("impl FromStr for Chain {\n");
    code.push_str("    type Err = InvalidChainError;\n\n");
    code.push_str("    fn from_str(s: &str) -> Result<Self, Self::Err> {\n");
    code.push_str("        match s {\n");
    code.push_str("            \"Any\" | \"any\" | \"ANY\" => Ok(Chain::Any),\n");

    let active_chains: Vec<_> = chains
        .iter()
        .filter(|c| c.status == ChainStatus::Active || c.status == ChainStatus::DevnetOnly)
        .collect();

    for chain in active_chains {
        let lower = chain.name_lower();
        let upper = chain.name.to_uppercase();

        // Handle special cases where lowercase/uppercase might differ significantly
        if chain.needs_special_case() {
            code.push_str(&format!(
                "            \"{}\" | \"{}\" => Ok(Chain::{}),\n",
                chain.name, lower, chain.name
            ));
        } else {
            code.push_str(&format!(
                "            \"{}\" | \"{}\" | \"{}\" => Ok(Chain::{}),\n",
                chain.name, lower, upper, chain.name
            ));
        }
    }

    code.push_str("            _ => {\n");
    code.push_str("                // Handle Unknown(n) format\n");
    code.push_str("                let mut parts = s.split(&['(', ')']);\n");
    code.push_str("                let _ = parts\n");
    code.push_str("                    .next()\n");
    code.push_str("                    .filter(|name| name.eq_ignore_ascii_case(\"unknown\"))\n");
    code.push_str("                    .ok_or_else(|| InvalidChainError(s.into()))?;\n\n");
    code.push_str("                parts\n");
    code.push_str("                    .next()\n");
    code.push_str("                    .and_then(|v| v.parse::<u16>().ok())\n");
    code.push_str("                    .map(Chain::from)\n");
    code.push_str("                    .ok_or_else(|| InvalidChainError(s.into()))\n");
    code.push_str("            }\n");
    code.push_str("        }\n");
    code.push_str("    }\n");
    code.push_str("}\n");

    code
}

fn main() {
    // Path to the Go source file (relative to workspace root)
    let go_source_path = PathBuf::from("../../vaa/structs.go");

    // Tell cargo to rerun if the Go source changes
    println!("cargo:rerun-if-changed={}", go_source_path.display());

    // Read the Go source
    let go_source = fs::read_to_string(&go_source_path)
        .unwrap_or_else(|e| panic!("Failed to read Go source at {:?}: {}", go_source_path, e));

    // Parse chain definitions
    let chains = parse_go_constants(&go_source);

    println!(
        "cargo:warning=Parsed {} chain definitions from Go source",
        chains.len()
    );

    // Generate Rust code
    let rust_code = generate_rust_code(&chains);

    // Write to OUT_DIR
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());
    let generated_file = out_dir.join("chains_generated.rs");

    fs::write(&generated_file, rust_code)
        .unwrap_or_else(|e| panic!("Failed to write generated code: {}", e));

    println!(
        "cargo:warning=Generated chains code at {:?}",
        generated_file
    );
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_active_chain() {
        let input = "    ChainIDSolana ChainID = 1";
        let result = parse_chain_constant(input);
        assert_eq!(result, Some(("Solana".to_string(), 1)));
    }

    #[test]
    fn test_parse_obsolete_chain() {
        let input = "    // OBSOLETE: ChainIDOasis ChainID = 7";
        let result = parse_obsolete_line(input.strip_prefix("    // OBSOLETE:").unwrap());
        assert_eq!(result, Some(("Oasis".to_string(), 7)));
    }

    #[test]
    fn test_extract_name() {
        assert_eq!(ChainDef::extract_name("ChainIDSolana"), "Solana");
        assert_eq!(ChainDef::extract_name("ChainIDBSC"), "BSC");
        assert_eq!(ChainDef::extract_name("ChainIDEthereum"), "Ethereum");
    }

    #[test]
    fn test_parse_go_constants() {
        let input = r#"
const (
    ChainIDUnset ChainID = 0
    ChainIDSolana ChainID = 1
    ChainIDEthereum ChainID = 2
    // OBSOLETE: ChainIDOasis ChainID = 7
    ChainIDAlgorand ChainID = 8
)
        "#;

        let chains = parse_go_constants(input);

        // Should have Unset, Solana, Ethereum, Oasis (obsolete), and Algorand
        assert_eq!(chains.len(), 5);

        // Check Solana
        let solana = chains.iter().find(|c| c.name == "Solana").unwrap();
        assert_eq!(solana.id, 1);
        assert_eq!(solana.status, ChainStatus::Active);

        // Check Oasis (obsolete)
        let oasis = chains.iter().find(|c| c.name == "Oasis").unwrap();
        assert_eq!(oasis.id, 7);
        assert_eq!(oasis.status, ChainStatus::Obsolete);
    }
}
