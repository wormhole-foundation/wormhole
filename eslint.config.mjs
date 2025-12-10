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
      "@typescript-eslint/strict-boolean-expressions": "error"
    },
  },
  includeIgnoreFile(resolve("./.gitignore"), "Imported .gitignore patterns"),
);