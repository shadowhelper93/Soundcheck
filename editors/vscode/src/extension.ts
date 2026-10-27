import * as vscode from 'vscode';
import { SoundcheckCLI } from './cli';
import { SoundcheckStatusBar } from './statusBar';
import { AdaptersTreeProvider, RolesTreeProvider, ActionsTreeProvider } from './treeView';
import { promptAndExecute } from './execCommand';
import { DashboardWebview } from './dashboardWebview';

export function activate(context: vscode.ExtensionContext): void {
  const outputChannel = vscode.window.createOutputChannel('Soundcheck Execution');
  const statusBar = new SoundcheckStatusBar();

  const adaptersProvider = new AdaptersTreeProvider();
  const rolesProvider = new RolesTreeProvider();
  const actionsProvider = new ActionsTreeProvider();

  vscode.window.registerTreeDataProvider('soundcheck.adaptersView', adaptersProvider);
  vscode.window.registerTreeDataProvider('soundcheck.rolesView', rolesProvider);
  vscode.window.registerTreeDataProvider('soundcheck.actionsView', actionsProvider);

  const runDoctor = async (silent = false): Promise<void> => {
    statusBar.setLoading();
    try {
      const report = await SoundcheckCLI.runDoctor();
      statusBar.updateReport(report);
      adaptersProvider.update(report.adapters);
      rolesProvider.update(report.resolutions);
      if (!silent) {
        vscode.window.showInformationMessage(
          `Soundcheck: ${report.summary.functionalAgents}/${report.summary.totalAgents} CLI adapters functional.`
        );
      }
    } catch (err: any) {
      statusBar.setError(err?.message || 'Check failed');
      if (!silent) {
        vscode.window.showErrorMessage(`Soundcheck Doctor Error: ${err?.message || err}`);
      }
    }
  };

  const runResolve = async (): Promise<void> => {
    try {
      const roles = await SoundcheckCLI.runResolve();
      rolesProvider.update(roles);
      vscode.window.showInformationMessage('Soundcheck: Roles resolved successfully.');
    } catch (err: any) {
      vscode.window.showErrorMessage(`Soundcheck Resolve Error: ${err?.message || err}`);
    }
  };

  const openTerminal = async (): Promise<void> => {
    const choice = await vscode.window.showQuickPick(
      ['claude', 'codex', 'opencode', 'agy', 'gemini', 'copilot'],
      { placeHolder: 'Select CLI adapter to launch in raw PTY terminal' }
    );
    if (!choice) return;

    const bin = SoundcheckCLI.resolveBinary();
    const term = vscode.window.createTerminal({
      name: `Soundcheck: ${choice}`,
      shellPath: bin,
      shellArgs: ['run', choice],
    });
    term.show();
  };

  const showStatusMenu = async (): Promise<void> => {
    const pick = await vscode.window.showQuickPick(
      [
        { label: '$(pulse) Run Doctor Preflight Check', cmd: 'soundcheck.doctor' },
        { label: '$(play) Execute Prompt with Auto-Failover', cmd: 'soundcheck.exec' },
        { label: '$(references) Resolve Agent Roles', cmd: 'soundcheck.resolve' },
        { label: '$(browser) Open Web Dashboard', cmd: 'soundcheck.openDashboard' },
        { label: '$(terminal) Open Interactive PTY Terminal', cmd: 'soundcheck.openTerminal' },
        { label: '$(refresh) Refresh Health Status', cmd: 'soundcheck.refresh' },
      ],
      { placeHolder: 'Soundcheck Multi-Agent Actions' }
    );
    if (pick) {
      vscode.commands.executeCommand(pick.cmd);
    }
  };

  context.subscriptions.push(
    statusBar,
    outputChannel,
    vscode.commands.registerCommand('soundcheck.doctor', () => runDoctor(false)),
    vscode.commands.registerCommand('soundcheck.resolve', runResolve),
    vscode.commands.registerCommand('soundcheck.exec', () => promptAndExecute(outputChannel)),
    vscode.commands.registerCommand('soundcheck.openDashboard', () => DashboardWebview.open(context)),
    vscode.commands.registerCommand('soundcheck.openTerminal', openTerminal),
    vscode.commands.registerCommand('soundcheck.refresh', () => runDoctor(false)),
    vscode.commands.registerCommand('soundcheck.statusMenu', showStatusMenu)
  );

  const autoDoctor = vscode.workspace.getConfiguration('soundcheck').get<boolean>('autoDoctorOnStartup');
  if (autoDoctor) {
    runDoctor(true);
  }
}

export function deactivate(): void {}
