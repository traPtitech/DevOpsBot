# DevOpsBot

ChatOps を行う bot です。

## 設定

設定はファイルを通して行います。

`CONFIG_FILE` 環境変数に設定されたパスから、設定ファイルを読み込みます。
指定が無い場合は `./config.yaml` をデフォルトで読み込みます。

### 設定ファイルの書き方

すべてのコマンドは、1つの「テンプレート」を通して実行されます。

`# コメント` となっている行は yaml のコメント行です。
Bot が読み取るファイルの中身には関係ありません。

```yaml
# テンプレート一覧
templates:
  - name: echo-template
    command: |
      #!/bin/sh
      
      echo test-arg1 "$@"

# 実際に実行できるコマンド一覧
commands:
    # (required) コマンドの名前 → この場合は /echo-test となる
  - name: echo-test
    # (required) templates で設定したテンプレートの名前
    templateRef: echo-template
    # (optional) /help で表示されるコマンドの説明
    description: "コマンドの説明"    
    # (optional) テンプレート実行時に、先頭に追加する引数
    argsPrefix:
      - test-arg2
      - test-arg3
    # (optional) テンプレートがユーザーからの引数をさらに必要とする場合、明示的に true と書く
    allowArgs: true
    # (optional) テンプレートがユーザーからの引数をさらに必要とする場合、ここにドキュメントを行う
    argsSyntax: "[example|extra|arg|description]"
    # (optional) このコマンド（とサブコマンド）を実行可能なユーザーの ID 一覧
    # 定義しなければ、全員がこのコマンド（とサブコマンド）実行可能になります
    operators:
      - toki
      - cp20
    # (optional) サブコマンドの定義 (フィールドは一緒)
    subCommands:
      - name: sub-command
        ... (省略)
```

以上の設定を反映し、`/echo-test test-arg4` と打つと、DevOpsBot のローカルで

- `echo test-arg1 test-arg2 test-arg3 test-arg4`

が実行されます。

テンプレートの中で SSH を使ったり、npm version と git push でバージョン更新を自動化したり、様々なスクリプトを実行できます。
