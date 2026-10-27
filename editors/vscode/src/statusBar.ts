import * as vscode from 'vscode';
import { DoctorReport } from './types';

export class SoundcheckStatusBar {
  private item: vscode.StatusBarItem;

  constructor() {
    this.item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    this.item.command = 'soundcheck.statusMenu';
    this.setInitialState();
    this.item.show();
  }

  public setInitialState(): void {
    this.item.text = '$(shield) Soundcheck: Ready';
    this.item.tooltip = 'Click to run Soundcheck preflight check';
    this.item.backgroundColor = undefined;
  }

  public setLoading(): void {
    this.item.text = '$(sync~spin) Soundcheck: Checking...';
    this.item.tooltip = 'Executing Soundcheck doctor preflight probes...';
  }

  public updateReport(report: DoctorReport): void {
    const total = report.summary.totalAgents;
    const functional = report.summary.functionalAgents;
    const isDegraded = functional < total;

    if (isDegraded) {
      this.item.text = `$(alert) Soundcheck: ${functional}/${total} Functional`;
      this.item.tooltip = `Soundcheck Preflight: DEGRADED\n${functional} of ${total} CLI adapters functional.\nClick to view details or fix.`;
      this.item.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
    } else {
      this.item.text = `$(shield) Soundcheck: ${functional}/${total} OK`;
      this.item.tooltip = `Soundcheck Preflight: HEALTHY\nAll ${total} CLI adapters functional.\n${report.summary.quirksProtected} quirks protected.\nClick for menu.`;
      this.item.backgroundColor = undefined;
    }
  }

  public setError(msg: string): void {
    this.item.text = '$(error) Soundcheck: Error';
    this.item.tooltip = `Soundcheck execution error: ${msg}`;
    this.item.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
  }

  public dispose(): void {
    this.item.dispose();
  }
}
