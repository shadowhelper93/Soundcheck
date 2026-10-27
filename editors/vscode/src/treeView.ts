import * as vscode from 'vscode';
import { Adapter, DetailedResolution } from './types';

export class AdaptersTreeProvider implements vscode.TreeDataProvider<AdapterTreeItem> {
  private _onDidChangeTreeData = new vscode.EventEmitter<AdapterTreeItem | undefined | void>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;
  private adapters: Adapter[] = [];

  public update(adapters: Adapter[]): void {
    this.adapters = adapters;
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: AdapterTreeItem): vscode.TreeItem {
    return element;
  }

  getChildren(element?: AdapterTreeItem): Thenable<AdapterTreeItem[]> {
    if (element) {
      // Sub-items for probe tiers
      const items: AdapterTreeItem[] = [];
      const tiers = ['L0', 'L1', 'L2', 'L3'];
      for (const tier of tiers) {
        const p = element.adapter.probes[tier];
        if (p) {
          const pass = p.status === 'passed';
          const icon = pass ? 'check' : p.status === 'failed' ? 'error' : 'dash';
          const label = `${tier}: ${p.message || p.status} (${p.latencyMs}ms)`;
          const item = new AdapterTreeItem(element.adapter, label, vscode.TreeItemCollapsibleState.None);
          item.iconPath = new vscode.ThemeIcon(icon);
          items.push(item);
        }
      }
      return Promise.resolve(items);
    }

    if (this.adapters.length === 0) {
      const placeholder = new vscode.TreeItem('No doctor data. Run doctor check.', vscode.TreeItemCollapsibleState.None);
      placeholder.iconPath = new vscode.ThemeIcon('info');
      return Promise.resolve([placeholder as any]);
    }

    return Promise.resolve(
      this.adapters.map((a) => {
        const l0 = a.probes.L0?.status === 'passed' ? '✓' : '✖';
        const l1 = a.probes.L1?.status === 'passed' ? '✓' : a.probes.L1?.status === 'failed' ? '✖' : '-';
        const l2 = a.probes.L2?.status === 'passed' ? '✓' : a.probes.L2?.status === 'failed' ? '✖' : '-';
        const l3 = a.probes.L3?.status === 'passed' ? '✓' : a.probes.L3?.status === 'failed' ? '✖' : '-';
        const meter = `[L0:${l0} L1:${l1} L2:${l2} L3:${l3}]`;

        const isOk = a.installed && a.probes.L1?.status === 'passed';
        const item = new AdapterTreeItem(
          a,
          `${a.id} · ${a.name} ${meter}`,
          vscode.TreeItemCollapsibleState.Collapsed
        );
        item.description = a.versionDetected || (a.installed ? 'installed' : 'missing');
        item.iconPath = new vscode.ThemeIcon(isOk ? 'pass' : a.installed ? 'warning' : 'circle-slash');
        return item;
      })
    );
  }
}

export class AdapterTreeItem extends vscode.TreeItem {
  constructor(
    public readonly adapter: Adapter,
    label: string,
    collapsibleState: vscode.TreeItemCollapsibleState
  ) {
    super(label, collapsibleState);
  }
}

export class RolesTreeProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
  private _onDidChangeTreeData = new vscode.EventEmitter<vscode.TreeItem | undefined | void>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;
  private roles: Record<string, DetailedResolution> = {};

  public update(roles: Record<string, DetailedResolution>): void {
    this.roles = roles;
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
    return element;
  }

  getChildren(): Thenable<vscode.TreeItem[]> {
    const keys = Object.keys(this.roles);
    if (keys.length === 0) {
      const placeholder = new vscode.TreeItem('No roles resolved. Click refresh.', vscode.TreeItemCollapsibleState.None);
      placeholder.iconPath = new vscode.ThemeIcon('info');
      return Promise.resolve([placeholder]);
    }

    return Promise.resolve(
      keys.map((role) => {
        const res = this.roles[role];
        const item = new vscode.TreeItem(
          `${role.toUpperCase()} ➔ ${res.selectedAgentId} (${res.selectedModel})`,
          vscode.TreeItemCollapsibleState.None
        );
        item.description = res.healthy ? 'HEALTHY' : 'UNHEALTHY';
        item.iconPath = new vscode.ThemeIcon(res.healthy ? 'verified' : 'unverified');
        item.tooltip = res.reason;
        return item;
      })
    );
  }
}

export class ActionsTreeProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
  getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
    return element;
  }

  getChildren(): Thenable<vscode.TreeItem[]> {
    const actions = [
      {
        label: 'Run Doctor Check',
        cmd: 'soundcheck.doctor',
        icon: 'pulse',
        tooltip: 'Run preflight health check across all CLI adapters',
      },
      {
        label: 'Execute Prompt (Auto-Failover)',
        cmd: 'soundcheck.exec',
        icon: 'play',
        tooltip: 'Run prompt through healthy resolved agent with auto-failover',
      },
      {
        label: 'Resolve Agent Roles',
        cmd: 'soundcheck.resolve',
        icon: 'references',
        tooltip: 'Evaluate role assignments against current adapter capabilities',
      },
      {
        label: 'Open Web Dashboard',
        cmd: 'soundcheck.openDashboard',
        icon: 'browser',
        tooltip: 'Open Soundcheck interactive web application',
      },
      {
        label: 'Open Interactive PTY Terminal',
        cmd: 'soundcheck.openTerminal',
        icon: 'terminal',
        tooltip: 'Open raw PTY session for an agent inside VS Code terminal',
      },
    ];

    return Promise.resolve(
      actions.map((act) => {
        const item = new vscode.TreeItem(act.label, vscode.TreeItemCollapsibleState.None);
        item.command = { command: act.cmd, title: act.label };
        item.iconPath = new vscode.ThemeIcon(act.icon);
        item.tooltip = act.tooltip;
        return item;
      })
    );
  }
}
