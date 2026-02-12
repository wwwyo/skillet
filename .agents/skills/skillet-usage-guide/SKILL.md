---
name: skillet-usage-guide
description: Skilletの使い方ガイド。スキル管理の初期化・同期・運用方法を説明する。
---

# Skillet 使い方ガイド

Skillet は AI エージェントのスキルを一元管理する CLI ツール。
複数の AI クライアント（Claude Code, Codex CLI）に分散するスキルを `.agents/` ディレクトリで Single Source of Truth として管理し、シンボリックリンクで各クライアントに同期する。

## インストール

```bash
go install github.com/wwwyo/skillet/cmd/skillet@latest
```

## 基本的なワークフロー

### 1. 初期化

```bash
# グローバルストアの初期化（~/.agents/ が作成される）
skillet init --global

# dotfiles リポジトリと連携する場合はパスを指定
skillet init -g --path ~/dotfiles/.agents

# プロジェクトストアの初期化（.agents/ が作成される）
skillet init --project
```

### 2. スキルの作成

スキルは `skills/` 配下にディレクトリを作り、`SKILL.md` を配置する。

```
# グローバルスキル（常時有効）
~/.agents/skills/<skill-name>/SKILL.md

# グローバルスキル（オプショナル）
~/.agents/skills/optional/<skill-name>/SKILL.md

# プロジェクトスキル（常時有効）
<project>/.agents/skills/<skill-name>/SKILL.md

# プロジェクトスキル（オプショナル）
<project>/.agents/skills/optional/<skill-name>/SKILL.md
```

`SKILL.md` のフォーマット:

```markdown
---
name: skill-name
description: スキルの説明
---

スキルの本文をここに記述する。
```

スキル名のルール:
- 英数字・ハイフン・アンダースコアのみ使用可能
- 先頭は英数字であること
- `/`, `\`, `..`, `.` で始まる名前は不可

### 3. 同期

```bash
# ドライランで変更内容を確認
skillet sync --dry-run

# 全ターゲットに同期
skillet sync

# 特定のターゲットのみ同期
skillet sync --target claude

# 強制同期（既存ファイルを上書き）
skillet sync --force
```

同期により `.claude/skills/` や `.codex/skills/` にシンボリックリンクが作成される。

### 4. ステータス確認

```bash
skillet status
```

各ターゲットのスキル同期状態（同期済み・未同期・余分なスキル）を一覧表示する。

### 5. スキルの一覧・削除

```bash
# 全スキルを一覧
skillet list

# スコープを指定して一覧
skillet list --global
skillet list --project

# スキルを削除
skillet remove <skill-name>
skillet remove <skill-name> --global
skillet remove <skill-name> --project
```

### 6. 既存スキルの移行

既に `.claude/skills/` や `.codex/skills/` に直接配置しているスキルがある場合、`.agents/` に移行できる。

```bash
# 対話形式で移行
skillet migrate

# 確認なしで移行
skillet migrate --yes

# スコープを指定して移行
skillet migrate --global
skillet migrate --project
```

## ディレクトリ構成

```
~/.config/skillet/
└── config.yaml              # グローバル設定

~/.agents/                   # グローバルスキルストア
└── skills/
    ├── skill-a/             # 常時有効スキル
    │   └── SKILL.md
    └── optional/            # オプショナルスキル
        └── skill-b/
            └── SKILL.md

<project>/
├── .agents/                 # プロジェクトスキルストア
│   ├── skills/
│   │   ├── skill-c/
│   │   │   └── SKILL.md
│   │   └── optional/
│   │       └── skill-d/
│   │           └── SKILL.md
│   └── skillet.yaml         # プロジェクト設定
├── .claude/
│   └── skills/              # symlink（gitignore 対象）
└── .codex/
    └── skills/              # symlink（gitignore 対象）
```

## 設定ファイル

### グローバル設定 (`~/.config/skillet/config.yaml`)

```yaml
version: 1
globalPath: ~/.agents          # グローバルスキルのパス
defaultStrategy: symlink       # symlink または copy

targets:
  claude:
    enabled: true
    globalPath: ~/.claude
  codex:
    enabled: true
    globalPath: ~/.codex
```

### プロジェクト設定 (`<project>/.agents/skillet.yaml`)

```yaml
version: 1

targets:
  claude:
    enabled: true
  codex:
    enabled: true
```

## スコープと優先度

同名のスキルが複数のスコープに存在する場合、優先度が高い方が使われる。

| スコープ | 場所 | 優先度 |
|----------|------|--------|
| Global | `~/.agents/skills/` | 低 |
| Project | `<project>/.agents/skills/` | 高 |

チーム共通のスキルをグローバルに配置し、プロジェクト固有のカスタマイズでオーバーライドする運用が推奨される。

## gitignore の設定

同期先ディレクトリは `.gitignore` に追加する:

```gitignore
.claude/skills/
.codex/skills/
```

`.agents/` ディレクトリ自体は Git で管理し、チームで共有する。
