# FR-25: 制限環境での worktree 作成

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了境界

AI agent のファイルアクセス制限により追跡ファイルの `.env`、`*.pem`、`.gitconfig` 等への書き込みが拒否される repository で、operator が global `ww` TOML に除外対象を設定して `ww create` を実行できるようにする。最初の checkout の前から除外し、対象ファイルに checkout の書き込みを試みない。実効設定が未指定または空なら従来どおり通常 checkout する。FR-25 全体の解決ではなく、その検証可能な一部を扱う。

共有 Git metadata の読み書き、refs、worktree 管理領域、Git 自体が必要とする config や hooks へのアクセスは sparse checkout では回避できない。外部 sandbox の制限を緩めずに済むとは約束しない。実環境での検証がなければ FR-25 を implemented にしない。

Addresses: FR-25 (`docs/project-plan.md`)

## 仕様と設計

- `checkout_exclude` 文字列配列を通常の設定フィールドとして追加する。global config を主な使用箇所とし、既存の設定規則どおり対象 project および repo-local の指定は配列全体を置換する。明示的な `[]` は除外を解除する。CLI flag は追加しない。Git の non-cone sparse-checkout pattern（gitignore 形式）の除外対象として扱い、repository 全体を含めた後で指定を除外する。`*.pem` と `.env` が全階層に適用される例、ルート限定の例、ディレクトリの例を説明する。空文字、改行、NUL、包含への反転など意図しないパターンは mutation 前に拒否する。Git のパターン解釈と TOML の引用を明記する。
- 通常・既存 branch・`--guess-remote` の各経路で、初回 checkout 前に sparse 設定が有効になるよう Git-native な順序を採用する。`git worktree add --no-checkout` の後に適用するだけでは、Git の共有 config や既存 main worktree の sparse 状態が変化し得る。Git 2.34.1 以降で実証し、既存 worktree と main の設定・展開内容を維持できない場合は実装を止め、制約と代替案を報告する。既存 sparse 設定を持つ repository も検証し、隔離不能な場合は checkout 前に明示的に失敗させる。
- 除外は checkout に限定し、設定済みの copy/symlink や lifecycle hook、Git hook、submodule が別途アクセスすることまで阻止しない。operator にこれらの設定見直しを案内する。自動権限変更、sandbox 回避、秘密ファイルの内容出力はしない。
- `--dry-run` は対象パターンと処理を表示し mutation、fetch、hook を実行しない。`--quiet` / `--json` の stdout 契約を守る。対話モードの既存挙動は維持する。
- 途中失敗時は元の Git エラーと失敗段階、残った branch/worktree/path および安全な確認・復旧方法を stderr に表示する。既存の所有権ベースの rollback 契約を維持し、登録済み worktree や不明な状態を勝手に削除しない。materialization と post-create hook は checkout 完了後だけ実行する。
- `.git/config`、worktree metadata、hooks や global config のアクセス拒否は対象ファイルの除外とは区別し、元エラーを保持した上で必要なアクセス権と手動復旧を案内する。外部制限を正確に分類できない場合は推測で断定しない。

## 変更箇所

- (MODIFY) `docs/specs/cli-commands.md` の create・失敗復旧・出力契約、`docs/specs/git-operations.md` の worktree 作成契約。仕様をコードより先に更新する。
- (MODIFY) `docs/specs/configuration.md` と利用者向け README に設定、アクセス権の前提と除外例を記載。`docs/project-plan.md` の FR-25 は残る制約と証拠に即して更新する。
- (MODIFY) `internal/config/config.go` の config field・layering、`cmd/ww` の manager 構築、`worktree/worktree.go` の config / `Manager.Create` / `createFailure`、`git/git.go` の worktree add ラッパーと checkout 支援。必要なら対象 worktree 用の runner を追加する。
- (MODIFY) `cmd/ww`、`git`、`worktree` の関連テストと `integration_test.go`。Git を使う結合テストで動作を検証する。

## 作業と検証

1. Git 2.34.1 と現行 Git で isolated fixture を用いて `--no-checkout`、worktree ごとの sparse 設定、初回 checkout の順序と共有 config・既存 sparse worktree への影響を検証する。`.git/config` 読み取り専用のケースも調べ、避けられない権限を確定する。実現不能なら互換性を偽装せず計画を見直す。
2. black-box 仕様と利用例を更新し、入力検証、各 branch 経路、dry-run、失敗時の残存状態を実装する。設定や引数を shell 展開せず Git に渡す。
3. 追跡された nested `.env`、`.pem`、`.gitconfig` と通常ファイルを持つ fixture で、除外ファイルが一度も作成されないことを検証する。global 設定、project/repo-local 置換、空配列、パターン、複数指定、無効値、通常動作、既存 branch、guess-remote、他 worktree の状態、hook と materialization の順序を検証する。checkout 前後の失敗を注入して元エラー、所有権保護、復旧案内、stdout を確認する。可能なら filesystem 拒否も再現し、模擬拒否と実際の sandbox の証拠を区別する。
4. `make fmt`、`make test-all`、`make lint`、`tools/test-workflow-lint.sh` と該当する workflow gate を実行する。実行環境や Git バージョンの制約があれば記録する。実 agent sandbox での手動検証は再現手順と結果を記録し、未実施なら未検証として残す。
5. 検証と PR 準備後にこの計画を削除し、実装 PR に結果と残る FR-25 の制約を記載する。

依存関係: 1 → 仕様 → 実装 → 検証。並行化する場合も共有 Git 状態の変更と失敗注入は fixture ごとに分離する。
