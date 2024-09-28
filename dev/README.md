# 開発環境

`make up`で開発環境が初期化されます。

- isucholar: `http://localhost:7000`
- Adminer: `http://localhost:7001`

`make down`で開発環境を終了します。

## 補足事項　WSL(Ubuntu)で動かす場合

- `sudo apt install gnupg2 pass` 左記のpackageが必要
- 社内からのdocker buildでつまづいたポイント。（centosだとpassがないから無理・・・？）
- ubuntuの際は時刻設定くるってたらapt-get出来ないので注意

