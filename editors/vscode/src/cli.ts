import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import { DoctorReport, DetailedResolution, RunPromptResult } from './types';

export class SoundcheckCLI {
  public static resolveBinary(): string {
    const configPath = vscode.workspace.getConfiguration('soundcheck').get<string>('binaryPath');
    if (configPath && fs.existsSync(configPath)) {
      return configPath;
    }

    const folders = vscode.workspace.workspaceFolders;
    if (folders && folders.length > 0) {
      const workspaceBin = path.join(folders[0].uri.fsPath, 'bin', 'soundcheck');
      if (fs.existsSync(workspaceBin)) {
        return workspaceBin;
      }
    }

    return 'soundcheck';
  }

  public static async runDoctor(): Promise<DoctorReport> {
    const stdout = await this.execute(['doctor', '--json']);
    return JSON.parse(stdout) as DoctorReport;
  }

  public static async runResolve(): Promise<Record<string, DetailedResolution>> {
    const stdout = await this.execute(['resolve', '--json']);
    return JSON.parse(stdout) as Record<string, DetailedResolution>;
  }

  public static async runExec(
    prompt: string,
    options?: { adapter?: string; role?: string }
  ): Promise<RunPromptResult> {
    const args = ['exec', '--json'];
    if (options?.adapter) {
      args.push('--adapter', options.adapter);
    } else if (options?.role) {
      args.push('--role', options.role);
    }
    args.push(prompt);

    const stdout = await this.execute(args);
    return JSON.parse(stdout) as RunPromptResult;
  }

  public static execute(args: string[]): Promise<string> {
    const bin = this.resolveBinary();
    const cwd = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath || process.cwd();

    return new Promise((resolve, reject) => {
      cp.execFile(
        bin,
        args,
        {
          cwd,
          maxBuffer: 10 * 1024 * 1024,
          env: {
            ...process.env,
            PATH: `${process.env.PATH}:/opt/homebrew/bin:/usr/local/bin:${process.env.HOME}/.local/bin:${process.env.HOME}/go/bin`,
          },
        },
        (error, stdout, stderr) => {
          if (error && !stdout) {
            reject(new Error(stderr || error.message));
            return;
          }
          resolve(stdout.trim());
        }
      );
    });
  }
}
