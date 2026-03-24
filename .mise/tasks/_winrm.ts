// WinRM SOAP client over fetch — no Python/pywinrm dependency.
// Usage:
//   import { WinRM } from "./_winrm.ts";
//   const winrm = new WinRM("127.0.0.1", 5985, "vagrant", "vagrant");
//   const result = await winrm.runPS("Get-Service sshd");
//   const result = await winrm.runCmd("dir");

const NS = {
  s: "http://www.w3.org/2003/05/soap-envelope",
  wsa: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
  wsman: "http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd",
  rsp: "http://schemas.microsoft.com/wbem/wsman/1/windows/shell",
};

const RESOURCE_CMD =
  "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd";

type CmdResult = { stdout: string; stderr: string; exitCode: number };

export class WinRM {
  private url: string;
  private auth: string;

  constructor(host: string, port: number, user: string, pass: string) {
    this.url = `http://${host}:${port}/wsman`;
    this.auth = btoa(`${user}:${pass}`);
  }

  // ── Low-level SOAP ────────────────────────────────────────────────────

  private envelope(action: string, body: string, shellId?: string): string {
    const selectors = shellId
      ? `<wsman:SelectorSet><wsman:Selector Name="ShellId">${shellId}</wsman:Selector></wsman:SelectorSet>`
      : "";
    return `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="${NS.s}" xmlns:wsa="${NS.wsa}" xmlns:wsman="${NS.wsman}" xmlns:rsp="${NS.rsp}">
  <s:Header>
    <wsa:To>${this.url}</wsa:To>
    <wsman:ResourceURI s:mustUnderstand="true">${RESOURCE_CMD}</wsman:ResourceURI>
    <wsa:ReplyTo><wsa:Address s:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</wsa:Address></wsa:ReplyTo>
    <wsa:Action s:mustUnderstand="true">${action}</wsa:Action>
    <wsman:MaxEnvelopeSize s:mustUnderstand="true">512000</wsman:MaxEnvelopeSize>
    <wsa:MessageID>uuid:${crypto.randomUUID()}</wsa:MessageID>
    <wsman:OperationTimeout>PT600S</wsman:OperationTimeout>
    ${selectors}
  </s:Header>
  <s:Body>${body}</s:Body>
</s:Envelope>`;
  }

  private async request(xml: string): Promise<string> {
    const res = await fetch(this.url, {
      method: "POST",
      headers: {
        "Content-Type": "application/soap+xml;charset=UTF-8",
        Authorization: `Basic ${this.auth}`,
      },
      body: xml,
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`WinRM HTTP ${res.status}: ${text.slice(0, 300)}`);
    }
    return res.text();
  }

  private async createShell(): Promise<string> {
    const body = `<rsp:Shell xmlns:rsp="${NS.rsp}">
      <rsp:InputStreams>stdin</rsp:InputStreams>
      <rsp:OutputStreams>stdout stderr</rsp:OutputStreams>
    </rsp:Shell>`;
    const res = await this.request(
      this.envelope("http://schemas.xmlsoap.org/ws/2004/09/transfer/Create", body),
    );
    const shellId = extractTag(res, "ShellId");
    if (!shellId) throw new Error("Failed to create WinRM shell");
    return shellId;
  }

  private async deleteShell(shellId: string): Promise<void> {
    await this.request(
      this.envelope("http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete", "", shellId),
    ).catch(() => {});
  }

  private async execCommand(
    shellId: string,
    command: string,
    args: string[] = [],
  ): Promise<CmdResult> {
    const argsXml = args.map((a) => `<rsp:Arguments>${a}</rsp:Arguments>`).join("");
    const body = `<rsp:CommandLine xmlns:rsp="${NS.rsp}">
      <rsp:Command>${escapeXml(command)}</rsp:Command>${argsXml}
    </rsp:CommandLine>`;
    const res = await this.request(
      this.envelope(
        "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command",
        body,
        shellId,
      ),
    );
    const commandId = extractTag(res, "CommandId");

    const recvBody = `<rsp:Receive xmlns:rsp="${NS.rsp}" SequenceId="0">
      <rsp:DesiredStream CommandId="${commandId}">stdout stderr</rsp:DesiredStream>
    </rsp:Receive>`;

    let stdout = "";
    let stderr = "";
    let exitCode = -1;

    while (true) {
      const recvRes = await this.request(
        this.envelope(
          "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive",
          recvBody,
          shellId,
        ),
      );
      const streams = extractStreams(recvRes);
      stdout += streams.stdout;
      stderr += streams.stderr;

      const exitMatch = recvRes.match(/ExitCode>(\d+)</);
      if (exitMatch) {
        exitCode = parseInt(exitMatch[1]);
        break;
      }
      if (recvRes.includes("CommandState") && recvRes.includes("Done")) {
        break;
      }
    }

    return { stdout, stderr, exitCode };
  }

  // ── Public API ────────────────────────────────────────────────────────

  /** Run a cmd.exe command */
  async runCmd(command: string): Promise<CmdResult> {
    const shellId = await this.createShell();
    try {
      return await this.execCommand(shellId, "cmd.exe", ["/c", command]);
    } finally {
      await this.deleteShell(shellId);
    }
  }

  /** Run a PowerShell script */
  async runPS(script: string): Promise<CmdResult> {
    // Encode as UTF-16LE base64 for powershell -EncodedCommand
    // Use Buffer to avoid stack overflow with large scripts (spread operator limit ~30K)
    const utf16 = Buffer.alloc(script.length * 2);
    for (let i = 0; i < script.length; i++) {
      utf16.writeUInt16LE(script.charCodeAt(i), i * 2);
    }
    const encoded = utf16.toString("base64");

    const shellId = await this.createShell();
    try {
      return await this.execCommand(shellId, "powershell.exe", [
        "-NoProfile",
        "-NonInteractive",
        "-EncodedCommand",
        encoded,
      ]);
    } finally {
      await this.deleteShell(shellId);
    }
  }

  /** Run PowerShell as SYSTEM via scheduled task (bypasses UAC) */
  async runElevated(psCode: string, timeout = 120): Promise<boolean> {
    const w = await this.runPS(
      `@'\n${psCode}\n'@ | Set-Content "C:\\bootstrap-step.ps1" -Force`,
    );
    if (w.exitCode !== 0) return false;

    await this.runPS(`
$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -File C:\\bootstrap-step.ps1"
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -RunLevel Highest
Register-ScheduledTask -TaskName "BootstrapStep" -Action $action -Principal $principal -Force | Out-Null
Start-ScheduledTask -TaskName "BootstrapStep"
`);

    for (let elapsed = 0; elapsed < timeout; elapsed += 10) {
      await Bun.sleep(10000);
      try {
        const r = await this.runPS(
          '(Get-ScheduledTask -TaskName "BootstrapStep" -ErrorAction SilentlyContinue).State',
        );
        if (r.stdout.trim() !== "Running") break;
      } catch {
        // WinRM may drop during heavy I/O (e.g. VS Build Tools install).
        // Keep polling — the task is still running inside the VM.
      }
    }

    try {
      await this.runPS(
        'Unregister-ScheduledTask -TaskName "BootstrapStep" -Confirm:$false -ErrorAction SilentlyContinue',
      );
      await this.runPS(
        'Remove-Item "C:\\bootstrap-step.ps1" -Force -ErrorAction SilentlyContinue',
      );
    } catch {
      // Best-effort cleanup — WinRM may still be recovering
    }
    return true;
  }

  /** Check if WinRM is reachable */
  async ping(timeoutMs = 3000): Promise<boolean> {
    try {
      await fetch(this.url, { signal: AbortSignal.timeout(timeoutMs) });
      return true;
    } catch {
      return false;
    }
  }
}

// ── XML helpers ───────────────────────────────────────────────────────────

function extractTag(xml: string, tag: string): string {
  const patterns = [
    new RegExp(`<[^>]*?${tag}[^>]*>([^<]*)<`, "i"),
    new RegExp(`<${tag}>([^<]*)<`, "i"),
  ];
  for (const re of patterns) {
    const m = xml.match(re);
    if (m) return m[1];
  }
  return "";
}

function extractStreams(xml: string): { stdout: string; stderr: string } {
  let stdout = "";
  let stderr = "";
  const re = /<(?:rsp:)?Stream[^>]*Name="(stdout|stderr)"[^>]*>([^<]*)<\/(?:rsp:)?Stream>/gi;
  let match;
  while ((match = re.exec(xml)) !== null) {
    // Decode base64 via Buffer to correctly handle UTF-8 (atob corrupts multi-byte chars)
    const decoded = Buffer.from(match[2], "base64").toString("utf-8");
    if (match[1] === "stdout") stdout += decoded;
    else stderr += decoded;
  }
  return { stdout: stdout.trim(), stderr: stderr.trim() };
}

function escapeXml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
