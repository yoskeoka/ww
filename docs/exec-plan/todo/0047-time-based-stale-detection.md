# 時間ベース stale 判定とグローバル既定値

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了境界

FR-23 を実装し、最終コミットから指定日数が経過した worktree を `stale` にする。
グローバル `ww` config で全体の既定日数を設定でき、設定がなければ時間判定を行わない。
時間ベースで stale になった worktree は `ww clean` の対象とし、**既定では dirty でも削除する**。
グローバル設定で dirty 保護へ戻せるようにする。

仕様更新、実装、境界値・設定優先順位・実削除を含むテスト、CLI help/README 更新、
品質ゲート、実装 PR の準備までが完了境界。実装 PR のマージは人間の承認を待つ。
この計画 PR 自体は機能実装や利用可能性を意味しない。

Addresses: N/A（ユーザー指定の FR-23。対応する外部 issue は指定なし）

## 確定したユーザー意図

- `--stale-days` と global config の全体既定値を提供する。
- 指定がなければ時間ベースの stale 判定をしない。
- 時間ベースの stale も既存同様に clean 対象に含める。
- 放置 worktree を片付ける用途なので dirty でも既定で消す。global config で調整可能にする。
- 「使われていない」は本 FR の最終コミット時刻で判定する。アクセス時刻やファイル更新時刻の追跡は追加しない。

## 現行実装の参照

参照元: ローカル `origin/main` の `ddf83870798afc08776f882746bb122d77d2db8f`。
ネットワーク復旧後に最新 main と照合し、差分があれば参照を更新する。

| 参照 | 現行の責務 |
| --- | --- |
| `docs/project-plan.md:69-81` | FR-23 と global config 関連要件 |
| `docs/specs/configuration.md:61-75,154-178` | 設定フィールド、global/project/local の置換順序 |
| `docs/specs/cli-commands.md` の `ww list` / `ww clean` / `ww i` | status、unknown、clean、対話操作の公開契約 |
| `docs/specs/git-operations.md` の Branch Operations | Git ベース判定・remote 確認・失敗時契約 |
| `docs/specs/interactive-mode.md:222-240` | list/clean と共通の cleanability |
| `docs/design-decisions/adr/0005-worktree-status-precedence.md:11-24` | merged 優先、main 除外、local-only の現行扱い。FR-23 を将来の例外として言及 |
| `docs/design-decisions/adr/0014-global-config-path-and-overrides.md:11-26` | global パスとフィールド単位の置換 |
| `internal/config/config.go:16-119,164-255` (`Config`, `configFields`, `decodeFields`, `mergeConfig`) | 値と明示指定を区別する設定レイヤー |
| `cmd/ww/main.go:140-214` (`newManagerWithOptions`) | internal config から公開 worktree.Config への変換 |
| `cmd/ww/helpers.go:39-111` (`managerForRepo`, `loadRepoConfigForSelection`) | 対象 repo の設定再解決 |
| `worktree/worktree.go:29-60,524-645,745-768` (`Config`, `Manager`, `listRepo`, `resolveStatus`) | 共通 status 判定・workspace の並列列挙 |
| `cmd/ww/sub_list.go:13-40,75-96` | CLI flags、表示、cleanable filter |
| `cmd/ww/sub_clean.go:24-74,77-156` | clean 候補と RemoveOpts の組立て |
| `cmd/ww/sub_interactive.go:91-153` | 対話 list/clean の共通 Manager と削除経路 |
| `worktree/worktree.go:69-76` (`RemoveOpts`) | dirty worktree 強制削除と branch 強制削除が現在は単一 Force |
| `integration_test.go:2675-2691` | merged/stale/active の既存 status 検証 |

## 公開動作の仕様

### 設定と CLI

| 項目 | 契約 |
| --- | --- |
| `stale_days` | TOML 整数。未設定または `0` は時間判定なし。正の値で有効。負数、型不正、扱えない大きさは明確なエラー |
| `stale_remove_dirty` | TOML bool。未設定の既定は `true`。時間ベース stale の clean 時に dirty worktree を削除可能にする。`false` なら従来の dirty 保護 |
| `ww list --stale-days N` | 当該呼出しの閾値を全対象 repo に適用する。明示 `0` は config の正数も上書きして無効化 |
| `ww clean --stale-days N` | list と同じ閾値指定。`--dry-run` / `--json` と併用可能 |
| `ww i --stale-days N` | 対話 list/clean に同じ明示 override を適用。設定値だけでも利用可能 |

両 TOML フィールドは既存の通常設定として global 共通、global `[[projects]]`、repo-local で利用可能。
優先順位は `CLI（stale_days のみ） > repo-local > 最初にマッチした global project > global 共通 > 未設定時既定`。
bool の明示 `false` と未設定を区別し、整数の明示 `0` と未設定も区別する。
materialization profile のフィールドには加えない。新しい `ww config` サブコマンドは追加しない。

workspace の list/clean/対話では各 repo の main root を基準に設定解決する。
呼出し元や最初の child の閾値・dirty policy を他 repo に流用しない。
CLI の `--stale-days` は列挙対象全体に適用し、削除用 Manager の再生成で失わない。
sandbox 時も既存の global パスと repo-local 境界を守る。

### 時間判定と status

1. 起点は各 worktree の HEAD commit の **committer timestamp**。author date、ファイル mtime、アクセス時刻、worktree 作成日時ではない。
2. `現在時刻 - committer timestamp >= N × 24時間` なら閾値到達。タイムゾーンや暦日切替に依存させない。
3. 同一 list/clean snapshot では共通の現在時刻を使用し、境界の前後を決定的に検証可能にする。日数換算で overflow させない。
4. main worktree、bare、branchless/detached entry は時間 stale にしない。
5. `merged` は従来どおり `stale` に優先する。未統合の local-only branch や remote が現存する branch も時間条件で stale にできる。
6. 既存の remote 消失 stale は時間設定の有無にかかわらず維持する。`--stale-days 0` は remote 判定を無効化しない。
7. base 判定不能時の `unknown(base-detect-failed)` は維持し、時間だけで clean 対象へ昇格させない。
8. 必要な timestamp の Git 取得/解析が失敗した場合は文脈付きエラーで列挙を失敗させ、削除を始めない。日時不明を古い値として扱わない。未来時刻は時間 stale にしない。
9. 時間判定無効時は timestamp 取得を追加しない。既存の remote cache は時間 stale を妨げず、時刻や最終 status を永続キャッシュしない。

`status` は既存の `active/merged/stale/unknown` を維持する。
時間理由と remote 理由を区別する additive JSON フィールド `stale_reason` を追加し、
時間条件に到達した `stale` では `time`、remote 消失だけなら `remote` を返す。
両方成立時は `time` とし dirty policy も時間経過に基づいて適用する。
stale 以外では省略する。既存 `status_detail=heuristic-base` はそのまま維持する。
text の STATUS 表示は既存形式のままとし、JSON と内部 snapshot で理由を保持する。

### dirty worktree の clean

| 候補と指定 | worktree 削除 | ローカル branch 削除 |
| --- | --- | --- |
| 時間理由の stale、`stale_remove_dirty` 未設定/true | dirty でも削除 | 従来の安全な `-d`。失敗なら branch を残して警告 |
| 時間理由の stale、`stale_remove_dirty = false` | dirty なら保護 | 従来どおり |
| merged、または remote 理由だけの stale | 従来の dirty 保護 | 従来どおり |
| 明示 `--force`（または対話の既存 force 選択） | dirty でも削除 | 従来の `-D` |

時間ベースの dirty 削除を実現するために既存 `RemoveOpts.Force` をそのまま自動 true にしない。
worktree の dirty 削除許可と branch 強制削除を区別する内部/公開 option を追加し、
既存 `Force` の意味・外部ライブラリ利用者の挙動は維持する。
時間理由・各 repo の policy を preview、JSON 結果、実削除、対話 snapshot の経路で保持する。
`ww clean --dry-run` はファイルや branch を消さず、dirty を消す予定であることを表示する。
対話 clean の最終確認でも dirty 削除の有効状態が分かるようにし、新しい承認段階は増やさない。
submodule 削除エラーの既存案内は維持し、Git が拒否した場合の再帰削除は導入しない。

## 変更マップ

- (MODIFY) `docs/specs/configuration.md`: フィールド、既定、検証、レイヤーの明示 0/false。
- (MODIFY) `docs/specs/cli-commands.md`: list/clean/i flags、status/reason、dirty 削除と preview。
- (MODIFY) `docs/specs/git-operations.md`: committer timestamp、取得失敗、dirty 削除と branch 削除の分離。
- (MODIFY) `docs/specs/interactive-mode.md`: 同一閾値・policy・最終確認の説明。
- (MODIFY) `docs/project-plan.md`: 実装検証完了時に FR-23 を implemented とし、global 既定と未設定時無効を追記。
- (MODIFY) `internal/config/config.go`, `internal/config/config_test.go`: decode/merge/validation と既定。
- (MODIFY) `worktree/worktree.go`, `worktree/worktree_test.go`: 公開 Config、時刻評価、理由、dirty 削除 option、既存 Force 互換。
- (MODIFY) `git/git.go`, `git/git_test.go`: HEAD committer timestamp の取得/解析。可能なら repo 単位でまとめ、worktree ごとの不要なプロセス増を避ける。
- (MODIFY) `cmd/ww/main.go`, `cmd/ww/helpers.go`: CLI override と repo ごとの設定解決。公開 worktree 層から internal/config は参照しない。
- (MODIFY) `cmd/ww/sub_list.go`, `cmd/ww/sub_clean.go`, `cmd/ww/sub_interactive.go`: flags、設定伝播、snapshot、dirty と branch force の分離。
- (MODIFY) `internal/interactive/` の該当 clean preview 表示とテスト: 既存最終確認へ dirty 削除状態を表示。
- (MODIFY) `integration_test.go`: 実 Git と隔離設定による公開動作の検証。
- (MODIFY) `README.md`: global `stale_days = 30` の例、`stale_remove_dirty`、CLI override、dirty の未コミット作業が削除される既定の説明。
- (DELETE) `docs/exec-plan/todo/0047-time-based-stale-detection.md`: 実装・検証完了後、実装 PR 準備時。

新しい ADR は必須としない。ADR 0005 が予告した FR-23 の例外を spec に明示し、受理済み ADR は書き換えない。

## 実行順序と依存

1. [ ] 最新 main へ同期し、承認・マージ済みの本計画から `feat/time-based-stale-detection` を `ww` で作る。
2. [ ] 上記の公開動作を specs に先に記述する。設定名、committer 時刻、0/false、dirty 既定と branch 保護を確認する。
3. [ ] config decode/validation/merge、公開 Config 変換、repo ごとの resolved policy を実装する。
4. [ ] Git timestamp 取得、共通時刻での時間判定、stale_reason を共通列挙へ実装する。
5. [ ] list/clean/i の override、preview/snapshot/実削除、dirty worktree と branch force の分離を接続する。
6. [ ] 以下の単体・integration 検証、help/README を整備する。
7. [ ] 品質ゲートを実行し、FR-23 更新、本計画削除、post-task-review、review-task、実装 PR と latest-head follow-up を行う。

順序は 2 → 3/4 → 5 → 6 → 7。config と timestamp の実装は仕様確定後に独立に進められるが、
共通ファイルの競合が多いため単一担当でよい。sub-agent 使用は要求しない。

## 検証

### 単体・設定

- 時刻固定で N 日直前、ちょうど、直後、未来時刻、author/committer 日時差、大きすぎる日数を検証。
- 未設定、global 正数、project 上書き、local 上書き、明示 0、CLI 0/正数、負数・型不正を検証。
- `stale_remove_dirty` の未設定 true、global false、project/local の true/false 上書きを検証。
- merged 優先、main/bare/detached 除外、local-only と remote 現存時の時間 stale、remote 消失既存動作、unknown 維持、timestamp 失敗を検証。
- 無効時に timestamp 取得を呼ばないこと、既存の path-only 列挙を重くしないことを確認。

### CLI integration / 手動相当検証

隔離した global config、実 Git repo、過去の `GIT_COMMITTER_DATE` を使い sleep なしで確認する。

- config なしでは古い未統合 branch は active。global `stale_days = 30` だけで stale になり、JSON `stale_reason=time` が出る。
- `list --cleanable` と `clean --dry-run` の候補が一致し、`--stale-days 0` で時間候補が消える。
- old/young、tracking/local-only、異なる設定の child 2 repo を含む workspace、非 Git workspace root、secondary worktree 起動を検証。
- **dirty な時間 stale** に対し、既定の clean で tracked/untracked の未コミット変更を含む worktree が削除される。
- 同じ条件で `stale_remove_dirty = false` なら削除を拒否する。明示 `--force` はその保護を上書きする。
- dirty な merged と remote-only stale は時間対象でない限り既存どおり保護される。
- 自動 dirty 削除では未統合 branch が `-D` で消えないこと、明示 force では従来どおり消えることを実 Git で検証。
- dry-run は dirty ファイル・branch とも残し、実行予定の dirty 削除を説明する。JSON/対話でも同じ候補・policy を使う。
- timestamp 取得失敗時に clean が削除を始めないことを確認。

### 実装時の品質ゲート

```sh
rtk make fmt
rtk make lint
rtk make test-all
rtk make build
rtk ./tools/workflow-lint.sh --mode=pre-push
```

計画のみの PR は Markdown の構造、参照パス、plan/branch 対応、workflow lint を検証する。
Go 実装変更がない段階で機能テストの成功を主張しない。

## この計画の作業環境と復旧記録

- 通常操作: cwd `/home/yoske/src/github.com/yoskeoka/vibe-coding-workspace/ww` で
  `rtk ww create --repo ww plan/time-based-stale-detection` を実行。
- 期待: 最新 main を起点に workspace 配下へ計画 worktree を作成。
- 実結果: `.git/refs/heads/plan/time-based-stale-detection.lock` の作成が `Read-only file system` で失敗。
  `git fetch origin` も `.git/FETCH_HEAD` が read-only で失敗し、escalation でも変わらなかった。
- 復旧: 書込み可能な `/tmp/ww-fr23-plan` に `git clone --no-hardlinks` で複製し、origin を GitHub URL へ戻し、
  `rtk ww --sandbox create plan/time-based-stale-detection` で計画 worktree を作成。
- worktree: `/tmp/ww-fr23-plan/.worktrees/ww-fr23-plan@plan-time-based-stale-detection`。
- 影響: 元 checkout に実装変更なし。GitHub fetch は DNS 解決失敗で、escalation でも接続不能。
  最新 remote SHA の確認と push/PR 作成はネットワーク復旧後に必要。
