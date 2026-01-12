// @ts-check

import eslint from '@eslint/js';
import { defineConfig } from 'eslint/config';
import { includeIgnoreFile } from "@eslint/compat";
import tseslint from 'typescript-eslint';
import { resolve } from "path";

export default defineConfig(
  eslint.configs.recommended,
  tseslint.configs.recommendedTypeChecked,
  tseslint.configs.strictTypeChecked,
  {
    languageOptions: {
      parserOptions: {
        projectService: true,
      },
    },
  },
  {
    rules: {
      "@typescript-eslint/strict-boolean-expressions": "error",
      "@typescript-eslint/restrict-template-expressions": ["error", { "allowNumber": true }],
      "@typescript-eslint/no-unnecessary-condition": ["error", { "allowConstantLoopConditions": "only-allowed-literals" }],
    },
  },
  includeIgnoreFile(resolve("./.gitignore"), "Imported .gitignore patterns"),
);