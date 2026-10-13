#!/usr/bin/env node
// Pack the GoReleaser archives into npm packages: one per platform carrying the native `vern`
// binary, plus the @grosspoetrysystems/vernacular wrapper that resolves and runs it.

import { execFileSync } from "node:child_process";
import { chmodSync, copyFileSync, mkdirSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { basename, join, resolve } from "node:path";

const version = process.argv[2];
const releaseDir = resolve(process.argv[3] ?? ".release");
if (!version) {
  console.error("usage: node scripts/package-npm.mjs <version> [release-directory]");
  process.exit(2);
}

const scope = "@grosspoetrysystems";
const repository = { type: "git", url: "git+https://github.com/grosspoetrysystems/vernacular.git" };
const packedDir = join(releaseDir, "npm");
const workDir = join(packedDir, "packages");
const platforms = [
  { goos: "darwin", goarch: "arm64", os: "darwin", cpu: "arm64", binary: "vern" },
  { goos: "darwin", goarch: "amd64", os: "darwin", cpu: "x64", binary: "vern" },
  { goos: "linux", goarch: "arm64", os: "linux", cpu: "arm64", binary: "vern" },
  { goos: "linux", goarch: "amd64", os: "linux", cpu: "x64", binary: "vern" },
  { goos: "windows", goarch: "arm64", os: "win32", cpu: "arm64", binary: "vern.exe" },
  { goos: "windows", goarch: "amd64", os: "win32", cpu: "x64", binary: "vern.exe" },
];

rmSync(packedDir, { recursive: true, force: true });
mkdirSync(workDir, { recursive: true });

const archives = readdirSync(releaseDir).filter((name) => name.endsWith(".tar.gz") || name.endsWith(".zip"));
const manifest = [];
const optionalDependencies = {};

for (const platform of platforms) {
  const suffix = `_${platform.goos}_${platform.goarch}${platform.goos === "windows" ? ".zip" : ".tar.gz"}`;
  const archive = archives.find((name) => name.endsWith(suffix));
  if (!archive) throw new Error(`missing release archive ending in ${suffix}`);

  const shortName = `vernacular-${platform.os}-${platform.cpu}`;
  const packageName = `${scope}/${shortName}`;
  const packageDir = join(workDir, shortName);
  mkdirSync(packageDir, { recursive: true });
  if (archive.endsWith(".zip")) {
    execFileSync("unzip", ["-q", join(releaseDir, archive), "-d", packageDir]);
  } else {
    execFileSync("tar", ["-xzf", join(releaseDir, archive), "-C", packageDir]);
  }

  for (const file of [platform.binary, "LICENSE"]) {
    if (!readdirSync(packageDir).includes(file)) throw new Error(`${archive} is missing ${file}`);
  }
  if (platform.binary === "vern") chmodSync(join(packageDir, platform.binary), 0o755);

  writeJson(join(packageDir, "package.json"), {
    name: packageName,
    version,
    description: `Vernacular native CLI for ${platform.os}-${platform.cpu}`,
    license: "MIT",
    os: [platform.os],
    cpu: [platform.cpu],
    files: [platform.binary, "LICENSE"],
    repository,
    publishConfig: { access: "public" },
  });

  optionalDependencies[packageName] = version;
  manifest.push(pack(packageDir, packageName, platform.binary));
}

const wrapperDir = join(workDir, "vernacular");
mkdirSync(join(wrapperDir, "bin"), { recursive: true });
copyFileSync("LICENSE", join(wrapperDir, "LICENSE"));
mkdirSync(join(wrapperDir, "skills", "vernacular"), { recursive: true });
copyFileSync("skills/vernacular/SKILL.md", join(wrapperDir, "skills", "vernacular", "SKILL.md"));
writeFileSync(
  join(wrapperDir, "bin", "vern.cjs"),
  `#!/usr/bin/env node
const { spawnSync } = require("node:child_process");

const key = process.platform + "-" + process.arch;
const packages = ${JSON.stringify(Object.fromEntries(platforms.map((platform) => [`${platform.os}-${platform.cpu}`, `${scope}/vernacular-${platform.os}-${platform.cpu}`])), null, 2)};
const packageName = packages[key];
if (!packageName) {
  console.error("Vernacular does not support " + key + ".");
  process.exit(1);
}

const binaryName = process.platform === "win32" ? "vern.exe" : "vern";
let binary;
try {
  binary = require.resolve(packageName + "/" + binaryName);
} catch {
  console.error(
    "Vernacular's " + key + " package is missing. Remove node_modules and the package lockfile, then reinstall; npm can omit platform optional dependencies when a lockfile was generated on another platform."
  );
  process.exit(1);
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
chmodSync(join(wrapperDir, "bin", "vern.cjs"), 0o755);
writeFileSync(
  join(wrapperDir, "README.md"),
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
writeJson(join(wrapperDir, "package.json"), {
  name: `${scope}/vernacular`,
  version,
  description: "Private companion spaces for AI-assisted working material around a Git repository",
  license: "MIT",
  bin: { vern: "bin/vern.cjs" },
  engines: { node: ">=22" },
  files: ["bin", "skills", "LICENSE", "README.md"],
  optionalDependencies,
  repository,
  homepage: "https://github.com/grosspoetrysystems/vernacular",
  bugs: { url: "https://github.com/grosspoetrysystems/vernacular/issues" },
  publishConfig: { access: "public" },
});
manifest.push(pack(wrapperDir, `${scope}/vernacular`));

writeJson(join(packedDir, "manifest.json"), manifest);
console.log(`Packed ${manifest.length} npm packages in ${packedDir}`);

function pack(packageDir, packageName, binary) {
  const output = JSON.parse(
    execFileSync("npm", ["pack", packageDir, "--pack-destination", packedDir, "--ignore-scripts", "--json"], {
      encoding: "utf8",
    }),
  );
  if (output.length !== 1) throw new Error(`npm pack returned ${output.length} results for ${packageName}`);
  return {
    name: packageName,
    version,
    file: basename(output[0].filename),
    directory: `packages/${basename(packageDir)}`,
    ...(binary ? { binary } : {}),
  };
}

function writeJson(path, value) {
  writeFileSync(path, `${JSON.stringify(value, null, 2)}\n`);
}
