import * as path from "path";
import * as fs from "fs";
import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export function activate(context: vscode.ExtensionContext) {
  const config = vscode.workspace.getConfiguration("sqlm");

  const serverPath = config.get<string>("serverPath") || "sqlm";
  const entitiesDir = resolveEntitiesDir(config.get<string>("entitiesDir"));

  if (!entitiesDir) {
    vscode.window.showWarningMessage(
      "sqlm: could not find entities directory. Set sqlm.entitiesDir in settings."
    );
    return;
  }

  const serverOptions: ServerOptions = {
    command: serverPath,
    args: ["lsp", entitiesDir],
    transport: TransportKind.stdio,
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "sqlm" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.sqlm"),
    },
  };

  client = new LanguageClient(
    "sqlm",
    "sqlm Language Server",
    serverOptions,
    clientOptions
  );

  client.start();
  context.subscriptions.push(client);
}

export function deactivate(): Thenable<void> | undefined {
  return client?.stop();
}

// resolveEntitiesDir returns the configured or auto-detected entities directory.
function resolveEntitiesDir(configured: string | undefined): string | undefined {
  if (configured) {
    return configured;
  }

  const workspaceFolders = vscode.workspace.workspaceFolders;
  if (!workspaceFolders) {
    return undefined;
  }

  for (const folder of workspaceFolders) {
    const candidate = path.join(
      folder.uri.fsPath,
      "migrations",
      "entities"
    );
    if (fs.existsSync(path.join(candidate, "main.sqlm"))) {
      return candidate;
    }
  }

  return undefined;
}
