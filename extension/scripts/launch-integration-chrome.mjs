import puppeteer from "puppeteer-core";

const [executablePath, extensionPath, userDataDir] = process.argv.slice(2);
if (!executablePath || !extensionPath || !userDataDir) {
  throw new Error("usage: launch-integration-chrome <chrome> <extension> <profile>");
}

const browser = await puppeteer.launch({
  executablePath,
  userDataDir,
  headless: true,
  pipe: true,
  enableExtensions: [extensionPath],
  args: [
    "--no-first-run",
    "--no-default-browser-check",
    "--disable-background-networking",
    "--disable-component-update",
    "--disable-sync",
    "--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE localhost",
  ],
});

const workerTarget = await browser.waitForTarget(
  (target) => target.type() === "service_worker" && target.url().endsWith("/dist/service-worker.js"),
  { timeout: 30_000 },
);
const worker = await workerTarget.worker();
worker?.on("console", (message) => process.stderr.write(`[extension] ${message.text()}\n`));
process.stdout.write(`READY ${workerTarget.url()}\n`);

async function stop() {
  await browser.close();
  process.exit(0);
}
process.stdin.resume();
process.stdin.once("end", () => void stop());
process.once("SIGTERM", () => void stop());
process.once("SIGINT", () => void stop());
