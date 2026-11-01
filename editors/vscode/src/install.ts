import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as crypto from 'crypto';
import * as fs from 'fs/promises';
import * as os from 'os';
import * as path from 'path';
import { promisify } from 'util';

const run = promisify(cp.execFile);
const releases = 'https://api.github.com/repos/shadowhelper93/Soundcheck/releases/latest';

async function download(url: string, destination: string): Promise<void> {
  if (process.platform === 'win32') {
    await run('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command',
      'Invoke-WebRequest -Uri $args[0] -OutFile $args[1]', url, destination], { timeout: 120000 });
  } else {
    await run('curl', ['-fLsS', '--max-time', '120', '-o', destination, url], { timeout: 125000 });
  }
}

export async function installCLI(context: vscode.ExtensionContext): Promise<void> {
  const platforms = [{ label: 'macOS', value: 'darwin' }, { label: 'Linux', value: 'linux' }, { label: 'Windows', value: 'win32' }];
  const selected = await vscode.window.showQuickPick(platforms, { placeHolder: `Select the OS where the extension runs (detected: ${os.platform()})` });
  if (!selected) return;
  if (selected.value !== process.platform) throw new Error('Select the detected operating system to install on this machine.');
  const arch = process.arch === 'x64' ? 'amd64' : process.arch === 'arm64' ? 'arm64' : '';
  if (!arch) throw new Error(`Unsupported architecture: ${process.arch}`);
  const response = await fetch(releases, { headers: { 'User-Agent': 'soundcheck-vscode', Accept: 'application/vnd.github+json' }, signal: AbortSignal.timeout(15000) });
  if (!response.ok) throw new Error(`No published Soundcheck release is available (GitHub HTTP ${response.status}).`);
  const release = await response.json() as { tag_name: string; assets: { name: string; browser_download_url: string }[] };
  const platform = selected.value === 'win32' ? 'windows' : selected.value;
  const name = `soundcheck_${release.tag_name.replace(/^v/, '')}_${platform}_${arch}.${platform === 'windows' ? 'zip' : 'tar.gz'}`;
  const archive = release.assets.find(a => a.name === name);
  const checksums = release.assets.find(a => a.name === 'checksums.txt');
  if (!archive || !checksums || ![archive, checksums].every(a => a.browser_download_url.startsWith('https://github.com/shadowhelper93/Soundcheck/releases/download/'))) {
    throw new Error(`Release ${release.tag_name} lacks the verified ${name} archive or checksums.`);
  }
  await fs.mkdir(context.globalStorageUri.fsPath, { recursive: true });
  const temp = await fs.mkdtemp(path.join(context.globalStorageUri.fsPath, 'install-'));
  try {
    const archivePath = path.join(temp, name);
    const checksumPath = path.join(temp, 'checksums.txt');
    await download(archive.browser_download_url, archivePath);
    await download(checksums.browser_download_url, checksumPath);
    const lines = (await fs.readFile(checksumPath, 'utf8')).split(/\r?\n/);
    const line = lines.find(l => l.endsWith(`  ${name}`) || l.endsWith(` *${name}`));
    const expected = line?.slice(0, 64);
    const actual = crypto.createHash('sha256').update(await fs.readFile(archivePath)).digest('hex');
    if (!expected || !/^[a-f0-9]{64}$/i.test(expected) || expected.toLowerCase() !== actual) throw new Error('Release checksum verification failed.');
    const binary = platform === 'windows' ? 'soundcheck.exe' : 'soundcheck';
    const extracted = await run(platform === 'windows' ? 'tar.exe' : 'tar', ['-xOf', archivePath, binary], { encoding: 'buffer', maxBuffer: 50 * 1024 * 1024, timeout: 30000 });
    const candidate = path.join(temp, binary);
    await fs.writeFile(candidate, extracted.stdout, { mode: 0o755 });
    await run(candidate, ['version'], { timeout: 10000 });
    const destination = path.join(context.globalStorageUri.fsPath, binary);
    await fs.copyFile(candidate, destination, fs.constants.COPYFILE_EXCL);
    await context.globalState.update('installedCLI', destination);
    vscode.window.showInformationMessage(`Soundcheck CLI ${release.tag_name} installed.`);
  } finally {
    await fs.rm(temp, { recursive: true, force: true });
  }
}
