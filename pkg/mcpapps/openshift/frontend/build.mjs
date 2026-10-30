import { build } from "esbuild";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const check = process.argv.includes("--check");

// Generate embedded assets, not a runtime web server. No external imports or
// font/image requests are needed when the host reads the MCP App resource.
const result = await build({
  absWorkingDir: fileURLToPath(new URL(".", import.meta.url)),
  entryPoints: ["main.jsx"],
  outfile: "../presentation.js",
  write: !check,
  bundle: true,
  format: "iife",
  jsx: "automatic",
  minify: true,
  legalComments: "inline",
  target: "es2022",
  define: { "process.env.NODE_ENV": '"production"' },
  loader: { ".svg": "dataurl", ".woff": "dataurl", ".woff2": "dataurl", ".ttf": "dataurl", ".eot": "dataurl" },
});

if (check) {
  for (const output of result.outputFiles) {
    let committed;
    try {
      committed = await readFile(output.path);
    } catch (error) {
      if (error.code !== "ENOENT") throw error;
    }
    if (!committed || !committed.equals(Buffer.from(output.contents))) {
      console.error(`${output.path} is missing or stale. Run make patternfly-assets and commit the regenerated bundles.`);
      process.exitCode = 1;
    }
  }
}
