import { execFileSync } from "node:child_process";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const EXPECTED_BUNDLE = ["LICENSE", "SKILL.md", "config.example.json"];

const repoRoot = new URL("..", import.meta.url).pathname;
const globalModules = execFileSync("npm", ["root", "-g"], { encoding: "utf8" }).trim();
const skillsModule = pathToFileURL(join(globalModules, "clawhub", "dist", "skills.js")).href;
const { listSkillFiles } = await import(skillsModule);

const actual = (await listSkillFiles(repoRoot)).map((file) => file.relPath).sort();
const expected = [...EXPECTED_BUNDLE].sort();

console.log(`ClawHub bundle: ${actual.join(", ")}`);
if (JSON.stringify(actual) !== JSON.stringify(expected)) {
  console.error(`Expected exactly: ${expected.join(", ")}`);
  console.error("Update .clawhubignore (and .claude/CLAWHUB_STANDARDS.md §9) deliberately if the bundle should change.");
  process.exit(1);
}
