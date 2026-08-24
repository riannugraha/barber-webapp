import { defineConfig } from "orval";

export default defineConfig({
  flowbook: {
    input: {
      target: "../api/openapi.yaml",
    },
    output: {
      mode: "single",
      target: "./generated/api.ts",
      client: "react-query",
      httpClient: "fetch",
      clean: false,
      prettier: true,
      override: {
        mutator: {
          path: "./lib/api.ts",
          name: "api",
        },
        query: {
          useQuery: true,
          useMutation: true,
        },
      },
    },
  },
});
