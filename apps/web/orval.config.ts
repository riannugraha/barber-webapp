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
      clean: true,
      prettier: true,
      override: {
        query: {
          useQuery: true,
          useMutation: true,
        },
      },
    },
  },
});
