#!/usr/bin/env node
// Pack the GoReleaser archives into one npm package, @grosspoetrysystems/vernacular, carrying every
// platform's native `vern` binary (about 3 MB each) under bin/<os>-<cpu>/, a launcher that runs
// the right one, and the agent skill.

import { execFileSync } from "node:child_process";
import { chmodSync, copyFileSync, cpSync, mkdirSync, mkdtempSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { basename, join, resolve } from "node:path";

const version = process.argv[2];
const releaseDir = resolve(process.argv[3] ?? ".release");
if (!version) {
  console.error("usage: node scripts/package-npm.mjs <version> [release-directory]");
  process.exit(2);
}

const name = "@grosspoetrysystems/vernacular";
const packedDir = join(releaseDir, "npm");
const packageDir = join(packedDir, "package");
const platforms = [
  { goos: "darwin", goarch: "arm64", key: "darwin-arm64", binary: "vern" },
  { goos: "darwin", goarch: "amd64", key: "darwin-x64", binary: "vern" },
  { goos: "linux", goarch: "arm64", key: "linux-arm64", binary: "vern" },
  { goos: "linux", goarch: "amd64", key: "linux-x64", binary: "vern" },
  { goos: "windows", goarch: "arm64", key: "win32-arm64", binary: "vern.exe" },
  { goos: "windows", goarch: "amd64", key: "win32-x64", binary: "vern.exe" },
];

rmSync(packedDir, { recursive: true, force: true });
mkdirSync(packageDir, { recursive: true });

const archives = readdirSync(releaseDir).filter((file) => file.endsWith(".tar.gz") || file.endsWith(".zip"));
for (const platform of platforms) {
  const suffix = `_${platform.goos}_${platform.goarch}${platform.goos === "windows" ? ".zip" : ".tar.gz"}`;
  const archive = archives.find((file) => file.endsWith(suffix));
  if (!archive) throw new Error(`missing release archive ending in ${suffix}`);

  const scratch = mkdtempSync(join(tmpdir(), "vern-npm-"));
  if (archive.endsWith(".zip")) {
    execFileSync("unzip", ["-q", join(releaseDir, archive), "-d", scratch]);
  } else {
    execFileSync("tar", ["-xzf", join(releaseDir, archive), "-C", scratch]);
  }
  if (!readdirSync(scratch).includes(platform.binary)) throw new Error(`${archive} is missing ${platform.binary}`);
  const target = join(packageDir, "bin", platform.key);
  mkdirSync(target, { recursive: true });
  copyFileSync(join(scratch, platform.binary), join(target, platform.binary));
  chmodSync(join(target, platform.binary), 0o755);
  rmSync(scratch, { recursive: true, force: true });
}

copyFileSync("LICENSE", join(packageDir, "LICENSE"));
cpSync("skills", join(packageDir, "skills"), { recursive: true });
writeFileSync(
  join(packageDir, "bin", "vern.cjs"),
  `#!/usr/bin/env node
const { chmodSync, existsSync } = require("node:fs");
const { join } = require("node:path");
const { spawnSync } = require("node:child_process");

const key = process.platform + "-" + process.arch;
const binary = join(__dirname, key, process.platform === "win32" ? "vern.exe" : "vern");
if (!existsSync(binary)) {
  console.error("Vernacular does not ship a binary for " + key + ".");
  process.exit(1);
}
if (process.platform !== "win32") {
  try {
    chmodSync(binary, 0o755);
  } catch {}
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
if (result.signal) process.kill(process.pid, result.signal);
process.exit(result.status ?? 1);
`,
);
chmodSync(join(packageDir, "bin", "vern.cjs"), 0o755);
writeFileSync(
  join(packageDir, "README.md"),
  `# Vernacular

Install the \`vern\` CLI:

\`\`\`sh
npm install --global @grosspoetrysystems/vernacular
\`\`\`

Supported platforms: macOS, Linux, and Windows on arm64 or x64. The agent skill is in
\`skills/vernacular/\` inside this package.

Documentation and source: https://github.com/grosspoetrysystems/vernacular
`,
);
writeJson(join(packageDir, "package.json"), {
  name,
  version,
  description: "Private companion spaces for AI-assisted working material around a Git repository",
  license: "MIT",
  bin: { vern: "bin/vern.cjs" },
  engines: { node: ">=22" },
  files: ["bin", "skills", "LICENSE", "README.md"],
  repository: { type: "git", url: "git+https://github.com/grosspoetrysystems/vernacular.git" },
  homepage: "https://github.com/grosspoetrysystems/vernacular",
  bugs: { url: "https://github.com/grosspoetrysystems/vernacular/issues" },
  publishConfig: { access: "public" },
});

const output = JSON.parse(
  execFileSync("npm", ["pack", packageDir, "--pack-destination", packedDir, "--ignore-scripts", "--json"], {
    encoding: "utf8",
  }),
);
if (output.length !== 1) throw new Error(`npm pack returned ${output.length} results`);
writeJson(join(packedDir, "manifest.json"), [
  { name, version, file: basename(output[0].filename), directory: "package" },
]);
console.log(`Packed ${name}@${version} (${Math.round(output[0].size / 1024)} KB) in ${packedDir}`);

function writeJson(path, value) {
  writeFileSync(path, `${JSON.stringify(value, null, 2)}\n`);
}
