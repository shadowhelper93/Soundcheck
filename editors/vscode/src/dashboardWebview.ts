import * as vscode from 'vscode';
import * as http from 'http';
import * as cp from 'child_process';
import { SoundcheckCLI } from './cli';

export class DashboardWebview {
  public static currentPanel: DashboardWebview | undefined;
  private readonly panel: vscode.WebviewPanel;
  private static serverProcess: cp.ChildProcess | undefined;

  public static async open(context: vscode.ExtensionContext): Promise<void> {
    const port = vscode.workspace.getConfiguration('soundcheck').get<number>('serverPort') || 3000;

    const running = await this.checkServer(port);
    if (!running) {
      this.startServer(port);
      // Wait for server to bind
      await new Promise((r) => setTimeout(r, 600));
    }

    if (DashboardWebview.currentPanel) {
      DashboardWebview.currentPanel.panel.reveal(vscode.ViewColumn.One);
      return;
    }

    const panel = vscode.window.createWebviewPanel(
      'soundcheckDashboard',
      'Soundcheck Dashboard',
      vscode.ViewColumn.One,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
      }
    );

    DashboardWebview.currentPanel = new DashboardWebview(panel, port);
  }

  private constructor(panel: vscode.WebviewPanel, port: number) {
    this.panel = panel;
    this.panel.webview.html = this.getHtml(port);

    this.panel.onDidDispose(() => {
      DashboardWebview.currentPanel = undefined;
    });
  }

  private static checkServer(port: number): Promise<boolean> {
    return new Promise((resolve) => {
      const req = http.get(`http://127.0.0.1:${port}/api/health`, (res) => {
        resolve(res.statusCode === 200);
      });
      req.on('error', () => resolve(false));
      req.setTimeout(500, () => {
        req.destroy();
        resolve(false);
      });
    });
  }

  private static startServer(port: number): void {
    if (this.serverProcess) {
      return;
    }
    const bin = SoundcheckCLI.resolveBinary();
    const cwd = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath || process.cwd();

    this.serverProcess = cp.spawn(bin, ['serve', '--port', String(port)], {
      cwd,
      detached: false,
      stdio: 'ignore',
    });

    this.serverProcess.on('exit', () => {
      DashboardWebview.serverProcess = undefined;
    });
  }

  private getHtml(port: number): string {
    const url = `http://127.0.0.1:${port}`;
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Soundcheck Dashboard</title>
  <style>
    body, html { margin: 0; padding: 0; width: 100%; height: 100%; overflow: hidden; background: #0a0a0f; }
    iframe { border: none; width: 100%; height: 100%; }
  </style>
</head>
<body>
  <iframe src="${url}"></iframe>
</body>
</html>`;
  }
}
