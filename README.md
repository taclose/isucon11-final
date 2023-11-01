# 練習メモ (taclose)

Macで`docker-compose`で環境を構築。Go言語でやってみた。その時の赤裸々なメモを残す。

## 初回計測 1646

とりあえずdockerでセットアップして、スローログ取れるようにしただけ。
細かいdocker設定変更は差分で見てほしい。ちょっと雑です。

このPCスペック低すぎないか。会社のパソコンだと１０倍は出たような...
ISUCON頑張ったら妻に改善を依頼しよう。

### Slow Query を調べた結果

```
# Profile
# Rank Query ID                            Response time Calls R/Call V/M
# ==== =================================== ============= ===== ====== ====
#    1 0xFFFCA4D67EA0A788813031B8BBC3B329  42.6701 49.5%  4834 0.0088  0.01 COMMIT
#    2 0xDA556F9115773A1A99AA0165670CE848  11.8051 13.7% 21393 0.0006  0.14 ADMIN PREPARE
#    3 0xDAFB520F3EA62E0FD8FCB26DC5D1E62F   3.7725  4.4%   212 0.0178  0.04 INSERT courses
#    4 0x7DFA2D5D9DBC803F79DB97773EC5447B   3.0094  3.5%  1688 0.0018  0.00 INSERT time_zone_transition
#    5 0x34680D6218DB2F97EAE350D366B60E94   2.1582  2.5%  2930 0.0007  0.00 INSERT unread_announcements
#    6 0x9304F82758390EA207A515217B0345EB   1.6896  2.0%     6 0.2816  0.47 INSERT users
#    7 0x02C5313CF776B3E250D6470BF2554CF6   1.4100  1.6%   138 0.0102  0.16 SELECT courses
#    8 0x9E2DA589A20EC24C34E11DDE0FBF5564   1.3178  1.5%  4912 0.0003  0.00 START
#    9 0x2BC75A9C3EB65F1EC570DFB4E3F111C5   1.3101  1.5%  1188 0.0011  0.00 SELECT announcements courses registrations unread_announcements
#   10 0x4A879C906EDA9FDE22E01B64DA82E819   1.0362  1.2%  2060 0.0005  0.00 SELECT registrations
#   11 0x93B0490C6027E6D67E3D4DA357148667   0.9806  1.1%  1789 0.0005  0.00 INSERT time_zone_transition_type
#   12 0xBCFA83825ED2F4DBAB748E30997E1B95   0.9721  1.1%   893 0.0011  0.09 INSERT UPDATE registrations
#   13 0x27543A651E60736B50443886E289E35A   0.9266  1.1%  1142 0.0008  0.00 UPDATE unread_announcements
#   14 0x1F21068CC7652980263E82315944F5FD   0.8401  1.0%  1248 0.0007  0.00 SELECT unread_announcements
#   15 0xF16955B9A50074ED04E1B8A511E35989   0.8219  1.0%  1144 0.0007  0.00 SELECT announcements courses unread_announcements
#   16 0x44EBC9974E0FE0B553B97C7B0FEFCE3D   0.7117  0.8%  1789 0.0004  0.00 INSERT time_zone_name
#   17 0x07890000813C4CC7111FD2D3F3B3B4EB   0.6965  0.8% 21393 0.0000  0.00 ADMIN CLOSE STMT
#   18 0x201AB42EC253BFB0F6BC1153F6B93083   0.6929  0.8%  1325 0.0005  0.00 SELECT courses
#   19 0xAA83D87B5B11FC07C50163FB3A1C4E9C   0.6565  0.8%  1789 0.0004  0.00 INSERT time_zone
#   20 0x6C0FF098D63D248BBF3D40588EFB42EF   0.5279  0.6%   432 0.0012  0.00 UPDATE submissions users
# MISC 0xMISC                               8.2478  9.6% 10805 0.0008   0.0 <81 ITEMS>
```

#### COMMOT / INSERTが多い　COMMITは42秒！！

initializeの処理ばっかり出てる・・・？INSERTとかばっかり。それともそういう処理が普通に多いのか？
であれば無駄なINDEXが多い可能性もあるのかもしれない。とりあえずあとでbenchmarkerの処理見てみよう。

#### ADMIN PREPAREが多い

DB接続時に`interpolateParams=true`とかすべきかも。

#### その他

time_zone_transitionって何？使った事ないけど、このシステムにタイムゾーンとかそんな特別な設定いるのか？
これは後々要らないって話になりそうな気もする。
announcementsとunread_announcementsがあるのか。別テーブルで管理という違和感はさておき、INSERT多いので要注意。

### ISUCHOLARの仕様を見てみる

- 優しい。各単語に英語を振ってくれていて、DBやコードの理解を助けてくれようとしている。
- 履修登録ページの検索機能から検索可能。重い処理ありそうだが..（予想で動くな、計測せよ）
- 講義が追加されると履修してる学生全員にunreadなお知らせが届くのか。ここは何かありそう。
- 同じ曜日で同じ時限のものは履修出来ないとか、あとでWHERE句要チェック（INDEX）
- 他にも主催者側の意図というかヒントが多い事がわかる内容だった

### BenchMarkerを読んでみる

- scenario/load.go を読むと、コメントが丁寧に振られていて、負荷のかけ方がわかるので見る価値はあるが、上級者用かもとも思った。
- ８時間の改善の中でここまで読む必要性があるかはちょっと現段階ではわからない。


## 



