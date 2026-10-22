# create 失敗時の新規ブランチ rollback

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: https://github.com/yoskeoka/ww/issues/294

## 目的と完了境界

通常の `ww create <branch>` が新規ブランチを作成した後、worktree 追加に失敗した場合、今回所有する未変更・未使用のブランチを自動で戻し、同名で再試行できるようにする。安全な rollback ができない場合は、元のエラーとともに残存ブランチ・worktree の状態、保留理由、対象 repo を明示した復旧手順を返す。

実装 PR の完了条件は、以下の仕様、失敗注入による回帰テスト、CLI 統合テスト、適用される品質ゲートの通過と、この計画の削除。計画 PR は Issue を閉じない。実装 PR に `Closes #294` を付ける。

本計画は FR-2 の失敗処理を修正する。FR-25 の全 sandbox 互換性、hook の副作用の復元、fetch の取り消し、強制終了後の回収、他プロセスの任意の Git 操作まで含む完全なトランザクションは完了境界に含めない。

## 報告とソースの照合

調査基準: `origin/main` = `51bf31ccdd0164760c79465f0b91150da72d6b8a`、Issue 本文とコメントなしの状態を 2026-10-22 に確認。

| 報告・観測 | 現行コードと意味 |
| --- | --- |
| upstream 設定時に Git が失敗するとブランチだけ残る | `git/git.go:77-81` の `Runner.WorktreeAdd` は `git worktree add -b` を直接実行する。Git の失敗は全変更の取り消しを意味しない。 |
| `ww` がそのまま exit 1 になる | `worktree/worktree.go:254-272` の `Manager.Create` は add エラーを返すだけで、残存状態の調査・rollback をしない。 |
| worktree がないため remove で回収できない | `worktree/worktree.go:790-822` の `Manager.Remove` は登録済み worktree を起点に検索し、見つからなければエラーになる。 |
| upstream は作成のためだけに使われている | `Manager.Create:269-270` は add 成功後に `BranchUnsetUpstream` を呼ぶ。その失敗も、作成済み worktree を残したままエラーになる。 |
| 既存ブランチや remote checkout と区別が必要 | `Manager.Create:207-218,254-264` は既存、新規通常、`--guess-remote` を分ける。`guessRemoteCreateError:350-365` には Git 非対応時の案内がある。 |
| 失敗時に成功結果を出してはいけない | `cmd/ww/sub_create.go:58-84` は Create のエラーを結果出力より先に返す。quiet/JSON でもこの性質を維持する。 |

### 一時 repo での確認

- ホストは Linux、Git 2.34.1、グローバル `ww` は Homebrew の 0.5.4。Git の system/global config と ww の config/cache を一時領域に分離した。
- `origin` の URL/fetch refspec、`origin/main`、`origin/HEAD`、`branch.autoSetupMerge=true` を設定し、`.git/config.lock` を置いた。remote 設定がない単なる `refs/remotes/origin/main` だけでは upstream 設定が起きず、失敗を再現しなかった。
- `git worktree add -b topic/reported <path> origin/main` は exit 255、`could not lock config file .git/config: File exists` と upstream 設定エラーを返した。ブランチは残り、対象 worktree は登録されなかった。
- 同じ repo の `ww create topic/ww-cli` は exit 1、stdout は空、同じ Git エラーを返し、ブランチだけ残った。
- config lock を維持したまま `git branch --no-track topic/split origin/main`、続く `git worktree add <path> topic/split` はともに成功した。
- さらに no-track ブランチ作成後、通常ファイルを親とするパスへの worktree add を失敗させた。作成時の完全 OID を条件にした `git update-ref -d refs/heads/<branch> <oid>` で、config を書かずに ref を削除できた。
- 調査時の一時 evidence は `/tmp/ww-294-repro-jr_kvrv6`。永続的な検証は、下記の自己完結した fixture を実装する。この一時ディレクトリに依存させない。
- config lock は Git の同じ失敗段階を再現する手段であり、報告元の macOS/Claude Code sandbox の `Operation not permitted` を直接再現した証拠ではない。

## 方針と代替案

ユーザー確認済み: **ブランチ作成と worktree 追加を分離し、安全に戻せる場合だけ自動削除する**。

1. 通常の新規作成は、base を完全 commit OID に解決し、`git branch --no-track <branch> <oid>` の成功後に `WorktreeAddExisting` を使う。base の追跡設定を作らず、成功後の upstream 解除も不要にする。
2. ブランチ作成コマンドの成功を今回の所有根拠にする。開始時に存在しなかったことだけでは、pre-create hook や別プロセスが作ったブランチの所有根拠にならない。同名作成が競合した場合は、作成失敗を返して相手のブランチに触らない。
3. add が失敗したら、今回所有する ref と作成時 OID、現在の完全 OID、worktree の登録状態、対象パスの状態を確認する。条件が揃う場合だけ、期待 OID 付きの ref 削除で rollback する。
4. `--guess-remote` は Git-native な解決・tracking の仕様を保つ。単一の Git コマンドが途中で作った ref の所有を証明できないため、失敗後の状態を調べて復旧案内を追加する。通常作成の所有根拠を流用して自動削除しない。

代替として「現行 `worktree add -b` のまま復旧案内のみ追加」を検討した。小さい変更だが、再現条件下で毎回手動回収が必要になるため選択しない。また `--no-track` を add に付けるだけでは別の add 失敗による残存を解決できない。開始前後の存在差だけで `branch -D` する案は、同名競合を誤って削除できるため採用しない。

既存の公開 `Runner.WorktreeAdd`、`BranchUnsetUpstream` の API を削除・変更しない。新しい操作の薄い wrapper を追加し、Manager の通常作成経路で組み合わせる。Git CLI 利用という既存設計内の修正とし、新しい ADR は作成しない。

## 実装前に更新する black-box 仕様

### `docs/specs/cli-commands.md` / create

- 通常の新規作成は base と同じ commit から、base を upstream にしないブランチを作る。上記 config-lock fixture では不要な config 書き込みを避け、他の制約がなければ作成に成功する。
- worktree add が失敗し、安全条件を満たす場合は今回の新規ブランチを取り消す。失敗は引き続き exit 1、元の Git エラーを保持する。再試行は既存ブランチ経路に誤って切り替わらない。
- 既存ブランチ、競合作成されたブランチ、作成後に更新・使用されたことを検知したブランチは削除しない。
- 対象パスまたは branch に結び付いた登録 worktree、対象パスの残存、状態取得の失敗、削除失敗があれば保全し、残存または確認不能の区別と次の確認・復旧コマンドを返す。
- branch 作成自体が失敗した場合も、部分的な ref 作成の可能性を調べて案内する。所有確認ができないものを削除しない。
- `--guess-remote` の add 失敗後も同じ残存診断を行う。remote 解決失敗と決めつけず、元のエラーを残し、既存の未対応 Git 向け案内を維持する。
- quiet/JSON を含め、失敗時に成功パスや成功 JSON を stdout に書かない。診断は stderr。失敗した add の後に materialization/post-create hook を実行しない。
- dry-run は branch/ref/config/worktree を変更せず、hook・fetch・rollback も実行しない。既存の remove/clean の動作には適用しない。

### `docs/specs/git-operations.md` / create・error handling

- `worktree add -b` → upstream 解除という通常作成の説明を、追跡設定なしの branch 作成 → 既存 branch への add に置き換える。
- 今回作成した branch のみを対象に、完全 OID を用いた条件付き削除、使用状況・パス確認、確認不能時の保全と診断を定義する。
- `.git/config` の書き込み権限回避を一般保証にしない。Git hooks、submodule、その他 Git の副作用は引き続き環境に依存する。

## 変更マップ

| 区分 | パス | 内容 |
| --- | --- | --- |
| MODIFY | `docs/specs/cli-commands.md` | create 失敗時の rollback、保全、診断、出力契約。 |
| MODIFY | `docs/specs/git-operations.md` | 通常作成の Git 操作と安全条件。 |
| MODIFY | `git/git.go` | no-track branch 作成、完全 OID とエラーを区別する ref 読み取り、期待 OID 付き削除の wrapper。既存公開 API を維持。 |
| MODIFY | `git/git_test.go` | 新規 wrapper の正常・失敗・OID 不一致・config-lock coverage。 |
| MODIFY | `worktree/worktree.go` | `Manager.Create` の分離、所有情報、rollback、復旧診断。 |
| MODIFY | `worktree/worktree_test.go` | 失敗注入と branch/worktree 保全、既存契約の回帰テスト。 |
| MODIFY | `integration_test.go` | 実 CLI の失敗・再試行、config lock、quiet/JSON、remote 診断。 |
| DELETE | `docs/exec-plan/todo/0046-create-failure-branch-rollback.md` | 実装検証と PR 準備後に削除する。 |

NEW のファイルは不要。`cmd/ww/sub_create.go` は現在のエラー伝播を利用し、変更前提にしない。

## 作業手順と依存関係

1. **仕様更新**: 上記 2 spec を先に変更する。実装開始時の main と Issue を再確認し、参照差分を取り込む。
2. **Git wrapper**: base を commit OID に固定し、その OID から no-track branch を作る操作を追加する。ref 不在と照会エラーを `BranchExists` の bool だけで判定しない。省略 OID の `WorktreeEntry.Head` は削除条件に使わない。
3. **通常作成の制御**: pre-create hook 後に branch 作成を行い、成功した branch/OID のみを所有情報として保持する。worktree add 成功後に materialization する。dry-run と既存 branch の順序・出力を維持する。
4. **rollback と診断**:
   - 元のエラーを保持したまま、ref が通常の local branch で完全 OID が一致すること、branch/対象パスに登録 worktree がないこと、対象パスが存在しないことを確認する。symlink と Lstat の確認エラーも「パスなし」と扱わない。
   - ref が消えていれば二度削除せず、元の失敗を返す。更新、不確かな所有、worktree/パス残存、照会失敗なら削除しない。
   - 削除は `git update-ref --no-deref -d refs/heads/<branch> <expected-oid>` 相当の条件付き操作にする。条件確認後の ref 更新も失敗させる。symbolic ref は自動削除対象にしない。config の一括削除や他 branch の変更をしない。
   - ref の条件付き削除と worktree 一覧は一つの Git transaction ではない。削除直前にも使用状況を確認し、任意の外部 Git checkout との完全な排他を保証すると記載しない。新たな lock 基盤を導入しない。
   - 登録 worktree や残ったパスの自動 remove、再帰削除、repo 全体の prune、remove hooks の実行は行わない。
   - rollback 不能時は、確認済み状態と確認できなかった状態を分ける。branch、worktree path、repo を明示し、`git -C <repo> worktree list --porcelain` と branch 確認、状態に合う復旧手順を shell-quote して出す。未確認の worktree に対する無条件の強制削除を案内しない。
   - 削除自体の失敗は元の add エラーに追記し、残存と cleanup エラーを隠さない。
5. **remote 失敗診断**: `WorktreeAddGuessRemote` のエラーにも残存確認と保全案内を付ける。fetch 失敗や unsupported Git など、ブランチを作っていない失敗との区別をテストする。
6. **検証と PR 準備**: 下記マトリクスと gates を実行し、計画を削除。`review-task` で PR 作成・最新 head の bounded follow-up を行う。

依存は 1 → 2 → 3 → 4/5 → 6。4 と 5 は診断 helper を共有するため Git write を直列化する。独立作業のための分割は不要。

## 検証マトリクス

既存参照: `worktree/worktree_test.go:610-739` の既存 branch/remote、`:773-801` の no-upstream、`:803-881` の hook、`git/git_test.go:721-775` の一時 repo helper、`integration_test.go:40-75` の CLI helper、`:872-995` の dry-run/quiet/JSON。既存 `Runner.GitBin` を使った fake Git の例は `worktree/worktree_test.go:689-739`。host-native fixture は ADR 0012 と `docs/specs/testing.md` に従う。

| ケース | 必須観測 |
| --- | --- |
| remote base + `.git/config.lock` | 通常 create 成功、期待 OID、upstream なし、lock/config の内容を変更しない。chmod に依存しない。 |
| no-track branch 成功後、add を mutation 前に失敗 | 元のエラー・exit 1、branch 不在、worktree/path 不在。同名再試行成功。post-create hook は未実行。 |
| branch 作成前に失敗 | 既存 ref/worktree に変更なし。残存確認結果に合う診断。 |
| pre-create hook/別プロセスが同名 branch を作成 | branch 作成が失敗し、競合 branch の OID と状態を保つ。 |
| add 失敗後、所有 branch が更新される | 更新された ref を保つ。比較確認後の更新も期待 OID 付き削除で拒否する。 |
| add が登録 worktree を残して失敗 / branch を別 worktree が使用 | branch/worktree/ファイルを保全し、残存場所と復旧案内。 |
| add が未登録パス、symlink を残す / Lstat が失敗 | パスを削除せず、branch を保全し、確認結果を案内。 |
| ref/worktree 照会失敗、削除失敗、symbolic ref | 強制削除せず、元のエラーと照会/cleanup の失敗を両方示す。 |
| ref が既に削除済み | 他の ref を変更せず、元の add エラーを保持。 |
| 既存 branch の add 失敗 | branch の OID/tracking を保持。新規 rollback を走らせない。 |
| `--guess-remote` が branch を残して失敗 | 自動削除せず残存診断。unsupported Git と missing remote の既存案内も検証。 |
| 通常/quiet/JSON で add 失敗 | stdout に成功結果なし、stderr に元のエラーと必要な復旧案内、exit 1。 |
| dry-run と成功経路 | mutation/rollback なし。既存 base 出力、既存 branch、remote tracking、copy/symlink/hook の契約が維持される。 |
| shell-quote が必要な repo/path/branch | 復旧コマンドが正しい対象を参照し、埋め込まれた文字列を実行しない。 |

失敗注入は実 Git へ委譲する小さな wrapper を使い、対象操作だけを決定的に失敗・更新させる。ユーザーの実 repo/config、実ネットワーク、Claude Code sandbox に依存させない。macOS の sandbox での実確認を行う場合は補足証拠とし、ホスト fixture の結果と混同しない。

実装後の gate（repo root）:

```bash
rtk make test
rtk make test-all
rtk make lint
rtk make build
rtk ./tools/workflow-lint.sh --mode=pre-push
```

計画 PR 自体は Markdown のみ。参照パス・行範囲、変更マップ、branch/plan 対応、Issue metadata、`git diff --check` と workflow lint を確認する。Go 実装の gate は実装 PR で実行する。
