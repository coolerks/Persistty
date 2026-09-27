import js from "@eslint/js";
import tseslint from "typescript-eslint";
import hooks from "eslint-plugin-react-hooks";
import refresh from "eslint-plugin-react-refresh";

export default tseslint.config(
  { ignores: ["dist", "node_modules"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  { files: ["scripts/*.mjs"], languageOptions: { globals: { console: "readonly" } } },
  {
    files: ["src/**/*.{ts,tsx}"],
    plugins: { "react-hooks": hooks, "react-refresh": refresh },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "error",
      "react-refresh/only-export-components": ["error", { allowConstantExport: true }],
      "@typescript-eslint/no-explicit-any": "error",
    },
  },
  { files: ["src/components/ui/*.tsx", "src/**/*.test.{ts,tsx}"], rules: { "react-refresh/only-export-components": "off" } },
);
