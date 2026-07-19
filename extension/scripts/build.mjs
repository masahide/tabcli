import { mkdir } from "node:fs/promises";
import { build } from "esbuild";

const outdir = new URL("../dist/", import.meta.url);
await mkdir(outdir, { recursive: true });

await build({
  entryPoints: {
    "service-worker": new URL("../src/service-worker.ts", import.meta.url).pathname,
    options: new URL("../src/options.ts", import.meta.url).pathname,
  },
  outdir: outdir.pathname,
  bundle: true,
  format: "esm",
  platform: "browser",
  target: "chrome121",
  sourcemap: false,
  legalComments: "none",
});
