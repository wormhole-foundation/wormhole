import { ParsedMoveToml } from "./types";

export class MoveToml {
  private toml: ParsedMoveToml;

  constructor(tomlStr: string) {
    this.toml = MoveToml.parse(tomlStr);
  }

  addRow(sectionName: string, key: string, value: string) {
    if (!MoveToml.isValidValue(value)) {
      if (/^\S+$/.test(value)) {
        value = `"${value}"`;
      } else {
        throw new Error(`Invalid value "${value}"`);
      }
    }

    const section = this.getSection(sectionName);
    section.rows.push({ key, value });
    return this;
  }

  getSectionNames(): string[] {
    return this.toml.map((s) => s.name);
  }

  isPublished(): boolean {
    return !!this.getRow("package", "published-at", false);
  }

  removeRow(sectionName: string, key: string) {
    const section = this.getSection(sectionName);
    section.rows = section.rows.filter((r) => r.key !== key);
    return this;
  }

  serialize(): string {
    let tomlStr = "";
    for (let i = 0; i < this.toml.length; i++) {
      const section = this.toml[i];
      tomlStr += `[${section.name}]\n`;
      for (const row of section.rows) {
        tomlStr += `${row.key} = ${row.value}\n`;
      }

      if (i !== this.toml.length - 1) {
        tomlStr += "\n";
      }
    }

    return tomlStr;
  }

  updateRow(sectionName: string, key: string, value: string) {
    if (!MoveToml.isValidValue(value)) {
      if (/^\S+$/.test(value)) {
        value = `"${value}"`;
      } else {
        throw new Error(`Invalid value "${value}"`);
      }
    }

    const row = this.getRow(sectionName, key);
    row.value = value;
    return this;
  }

  static isValidValue(value: string): boolean {
    value = value.trim();
    return (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("{") && value.endsWith("}")) ||
      (value.startsWith("'") && value.endsWith("'"))
    );
  }

  static parse(tomlStr: string): ParsedMoveToml {
    const toml: ParsedMoveToml = [];
    const lines = tomlStr.split("\n");
    for (const line of lines) {
      // Parse new section
      const sectionMatch = line.trim().match(/^\[(\S+)\]$/);
      if (sectionMatch && sectionMatch.length === 2) {
        toml.push({ name: sectionMatch[1], rows: [] });
        continue;
      }

      // Otherwise, parse row in section. We must handle two cases:
      //  1. value is string, e.g. name = "MyPackage"
      //  2. value is object, e.g. Sui = { local = "../sui-framework" }
      const rowMatch = line.trim().match(/^([a-zA-Z_\-]+) = (.+)$/);
      if (rowMatch && rowMatch.length === 3) {
        toml[toml.length - 1].rows.push({
          key: rowMatch[1],
          value: rowMatch[2],
        });
      }
    }

    return toml;
  }

  private getRow(
    sectionName: string,
    key: string,
    errorIfMissing: boolean = true
  ): ParsedMoveToml[number]["rows"][number] {
    const section = this.getSection(sectionName);
    const row = section.rows.find((r) => r.key === key);
    if (errorIfMissing && row === undefined) {
      throw new Error(`Row "${key}" not found in section "${sectionName}"`);
    }

    return row;
  }

  private getSection(
    sectionName: string,
    errorIfMissing: boolean = true
  ): ParsedMoveToml[number] {
    const section = this.toml.find((s) => s.name === sectionName);
    if (errorIfMissing && section === undefined) {
      throw new Error(`Section "${sectionName}" not found`);
    }

    return section;
  }
}
