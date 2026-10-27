import * as vscode from 'vscode';
import { SoundcheckCLI } from './cli';

export async function promptAndExecute(outputChannel: vscode.OutputChannel): Promise<void> {
  const prompt = await vscode.window.showInputBox({
    prompt: 'Enter prompt for AI agent execution',
    placeHolder: 'e.g. Write a unit test for auth middleware or say pong',
    ignoreFocusOut: true,
  });

  if (!prompt || prompt.trim() === '') {
    return;
  }

  const targetChoice = await vscode.window.showQuickPick(
    [
      { label: '$(sparkle) Default (Resolved Role Chain with Auto-Failover)', value: '' },
      { label: '$(account) Primary Role', value: 'primary', isRole: true },
      { label: '$(verified) Reviewer Role', value: 'reviewer', isRole: true },
      { label: 'Claude Code', value: 'claude' },
      { label: 'Antigravity CLI (AGY)', value: 'agy' },
      { label: 'OpenAI Codex', value: 'codex' },
      { label: 'OpenCode', value: 'opencode' },
      { label: 'Gemini CLI', value: 'gemini' },
      { label: 'Copilot CLI', value: 'copilot' },
    ],
    { placeHolder: 'Select target agent or role' }
  );

  if (targetChoice === undefined) {
    return;
  }

  const options: { adapter?: string; role?: string } = {};
  if (targetChoice.isRole) {
    options.role = targetChoice.value;
  } else if (targetChoice.value) {
    options.adapter = targetChoice.value;
  }

  await vscode.window.withProgress(
    {
      location: vscode.ProgressLocation.Notification,
      title: 'Soundcheck: Executing prompt with auto-failover...',
      cancellable: false,
    },
    async () => {
      try {
        const res = await SoundcheckCLI.runExec(prompt, options);
        outputChannel.clear();
        outputChannel.appendLine(`=======================================================`);
        outputChannel.appendLine(`SOUNDCHECK AGENT EXECUTION RESULT`);
        outputChannel.appendLine(`Prompt: "${res.prompt}"`);
        outputChannel.appendLine(`Agent Used: ${res.agentUsed} (${res.durationMs}ms)`);
        outputChannel.appendLine(`Success: ${res.success}`);
        outputChannel.appendLine(`Failover Occurred: ${res.failoverOccurred}`);

        if (res.failoverOccurred) {
          outputChannel.appendLine(`\nFailover Attempts:`);
          for (const att of res.attempts) {
            outputChannel.appendLine(`  - ${att.agentId}: ${att.success ? 'PASSED' : `FAILED (${att.errorMsg})`}`);
          }
          vscode.window.showInformationMessage(
            `Soundcheck: Auto-failover triggered! Final agent: ${res.agentUsed}. Check output channel.`
          );
        }

        outputChannel.appendLine(`\nOUTPUT:\n${res.output}`);
        outputChannel.appendLine(`=======================================================`);
        outputChannel.show(true);

        if (!res.success) {
          vscode.window.showErrorMessage(`Soundcheck: Prompt execution failed. See output channel.`);
        }
      } catch (err: any) {
        vscode.window.showErrorMessage(`Soundcheck Exec Error: ${err?.message || err}`);
      }
    }
  );
}
