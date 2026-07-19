import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { build } from "esbuild";

const outdir = new URL("../dist/", import.meta.url);
await mkdir(outdir, { recursive: true });

await build({
  entryPoints: {
    "service-worker": fileURLToPath(new URL("../src/service-worker.ts", import.meta.url)),
    options: fileURLToPath(new URL("../src/options.ts", import.meta.url)),
  },
  outdir: fileURLToPath(outdir),
  bundle: true,
  format: "esm",
  platform: "browser",
  target: "chrome121",
  sourcemap: false,
  legalComments: "none",
});
