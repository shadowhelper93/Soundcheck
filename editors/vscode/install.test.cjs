const assert = require('node:assert/strict');
const Module = require('node:module');
const os = require('node:os');
const path = require('node:path');
const fs = require('node:fs');
const esbuild = require('esbuild');
const crypto = require('node:crypto');
const cp = require('node:child_process');
const { promisify } = require('node:util');

async function main() {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'soundcheck-test-'));
  const output = path.join(temp, 'install.cjs');
  const originalLoad = Module._load;
  const originalFetch = global.fetch;
  const originalExec = cp.execFile;
  try {
    esbuild.buildSync({ entryPoints: [path.join(__dirname, 'src/install.ts')], outfile: output, bundle: true, platform: 'node', format: 'cjs', external: ['vscode'] });
    Module._load = function (name, ...args) {
      if (name === 'vscode') return { window: { showQuickPick: async options => options.find(o => o.value === process.platform), showInformationMessage: () => {} } };
      return originalLoad.call(this, name, ...args);
    };
    global.fetch = async () => ({ ok: false, status: 404 });
    let executeFake = async () => { throw new Error('unexpected command'); };
    const fakeExec = () => {};
    fakeExec[promisify.custom] = (...args) => executeFake(...args);
    cp.execFile = fakeExec;
    const { installCLI } = require(output);
    await assert.rejects(installCLI({ globalStorageUri: { fsPath: temp } }), /No published Soundcheck release/);
    assert.deepEqual(fs.readdirSync(temp), ['install.cjs']);

    const platform = process.platform === 'win32' ? 'windows' : process.platform;
    const arch = process.arch === 'x64' ? 'amd64' : process.arch;
    const name = `soundcheck_2.0.0_${platform}_${arch}.${platform === 'windows' ? 'zip' : 'tar.gz'}`;
    const base = 'https://github.com/shadowhelper93/Soundcheck/releases/download/v2.0.0/';
    global.fetch = async () => ({ ok: true, json: async () => ({ tag_name: 'v2.0.0', assets: [name, 'checksums.txt'].map(name => ({ name, browser_download_url: base + name })) }) });
    const fakeArchive = Buffer.from('archive');
    let validChecksum = false;
    executeFake = async (command, args) => {
      if (command === 'curl' || command === 'powershell.exe') {
        const dest = command === 'curl' ? args[args.indexOf('-o') + 1] : args[args.length - 1];
        const hash = validChecksum ? crypto.createHash('sha256').update(fakeArchive).digest('hex') : '0'.repeat(64);
        fs.writeFileSync(dest, dest.endsWith('checksums.txt') ? `${hash}  ${name}\n` : fakeArchive);
      }
      return { stdout: Buffer.from('binary'), stderr: Buffer.alloc(0) };
    };
    await assert.rejects(installCLI({ globalStorageUri: { fsPath: temp } }), /checksum verification failed/);
    assert.deepEqual(fs.readdirSync(temp), ['install.cjs']);
    validChecksum = true;
    let installed;
    await installCLI({ globalStorageUri: { fsPath: temp }, globalState: { update: async (_key, value) => { installed = value; } } });
    assert.equal(fs.readFileSync(installed, 'utf8'), 'binary');
    console.log('Installer rejects missing releases and invalid checksums; verified archives install.');
  } finally {
    Module._load = originalLoad;
    global.fetch = originalFetch;
    cp.execFile = originalExec;
    fs.rmSync(temp, { recursive: true, force: true });
  }
}
main().catch(err => { console.error(err); process.exitCode = 1; });
