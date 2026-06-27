import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

const plugins = [tailwindcss(), vue()];

// Instrument for coverage when COVERAGE=1 is set (make e2e-coverage).
// Keeps dev server fast for normal e2e runs.
if (process.env.COVERAGE === "1") {
  const istanbul = (await import("vite-plugin-istanbul")).default;
  plugins.push(
    istanbul({
      include: "src/**/*",
      exclude: ["node_modules", "dist"],
      extension: [".js", ".vue"],
      cypress: false,
      forceBuildInstrument: true,
    }),
  );
}

export default defineConfig({
  plugins,
  server: {
    port: parseInt(process.env.VITE_PORT || "5173"),
    proxy: {
      "/api": process.env.BACKEND_PORT
        ? `http://localhost:${process.env.BACKEND_PORT}`
        : process.env.API_PROXY_TARGET ||
          process.env.VITE_API_BASE ||
          "http://localhost:8080",
    },
  },
});
