# 練習メモ (taclose)

Macで`docker-compose`で環境を構築。Go言語でやってみた。その時の赤裸々なメモを残す。

## 初回計測 SCORE: 1646

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


## DB接続時に`interpolateParams=true` SCORE 2631

### Slow Query を調べた結果

```
# Profile
# Rank Query ID                            Response time Calls R/Call V/M
# ==== =================================== ============= ===== ====== ====
#    1 0xFFFCA4D67EA0A788813031B8BBC3B329  96.6388 65.1%  7359 0.0131  0.01 COMMIT
#    2 0x34680D6218DB2F97EAE350D366B60E94   5.2286  3.5%  4694 0.0011  0.02 INSERT unread_announcements
#    3 0xDAFB520F3EA62E0FD8FCB26DC5D1E62F   4.8062  3.2%   243 0.0198  0.00 INSERT courses
#    4 0x27543A651E60736B50443886E289E35A   3.6837  2.5%  2472 0.0015  0.02 UPDATE unread_announcements
#    5 0x7DFA2D5D9DBC803F79DB97773EC5447B   3.0615  2.1%  1688 0.0018  0.00 INSERT time_zone_transition
#    6 0xF16955B9A50074ED04E1B8A511E35989   2.8978  2.0%  2474 0.0012  0.00 SELECT announcements courses unread_announcements
#    7 0x4A879C906EDA9FDE22E01B64DA82E819   2.6617  1.8%  3584 0.0007  0.00 SELECT registrations
#    8 0x2BC75A9C3EB65F1EC570DFB4E3F111C5   2.4366  1.6%  1119 0.0022  0.00 SELECT announcements courses registrations unread_announcements
#    9 0x9E2DA589A20EC24C34E11DDE0FBF5564   2.3088  1.6%  7412 0.0003  0.00 START
#   10 0xBCFA83825ED2F4DBAB748E30997E1B95   1.3200  0.9%  1107 0.0012  0.02 INSERT UPDATE registrations
#   11 0x9304F82758390EA207A515217B0345EB   1.2597  0.8%     6 0.2099  0.34 INSERT users
#   12 0x1F21068CC7652980263E82315944F5FD   1.2392  0.8%  1179 0.0011  0.00 SELECT unread_announcements
#   13 0x63C05502F034B2DB20500EA0606444D4   1.1908  0.8%  1193 0.0010  0.02 SELECT classes
#   14 0x201AB42EC253BFB0F6BC1153F6B93083   1.1590  0.8%  1631 0.0007  0.00 SELECT courses
#   15 0x6C0FF098D63D248BBF3D40588EFB42EF   1.0891  0.7%   752 0.0014  0.00 UPDATE submissions users
#   16 0xD22122C68747429FE1B81D07D7E3DD52   1.0121  0.7%  1275 0.0008  0.00 SELECT courses
#   17 0x93B0490C6027E6D67E3D4DA357148667   0.9972  0.7%  1789 0.0006  0.00 INSERT time_zone_transition_type
#   18 0xD16D6453B62690656C8D51F2397F0B60   0.9811  0.7%   497 0.0020  0.05 INSERT announcements
#   19 0xAE994A27799DADD410E848351C4DFF53   0.9408  0.6%   776 0.0012  0.00 SELECT classes submissions
#   20 0x98DD1BA507AB79261637698A8004A814   0.9406  0.6%   765 0.0012  0.00 INSERT UPDATE submissions
# MISC 0xMISC                              12.6116  8.5% 15201 0.0008   0.0 <79 ITEMS>
```

`ADMIN PREPARE`は消えた。改善もした。だが`COMMITが相変わらず多い`

`Initialize`の処理を見たが、以下のSQLを実行していた。

```go
	files := []string{
		"1_schema.sql",
		"2_init.sql",
		"3_sample.sql",
	}
```

Table`unread_announcements` はsample.sqlで15行を１回のQUERYで呼んでるだけ。
これは逐次INSERTの度に`COMMIT`をしているパターンが疑わしい。
`AddAnnouncement` の処理のそこらへんを修正しよう。
※具体的な修正差分は差分を確認してください。

## unread_announcementsのINSERTを１回にまとめた（SCORE:2664)

### SLOW QUERY 確認

```
# Profile
# Rank Query ID                            Response time  Calls R/Call V/M
# ==== =================================== ============== ===== ====== ===
#    1 0xFFFCA4D67EA0A788813031B8BBC3B329  103.2055 67.8%  7845 0.0132  0.01 COMMIT
#    2 0xDAFB520F3EA62E0FD8FCB26DC5D1E62F    4.6383  3.0%   241 0.0192  0.01 INSERT courses
#    3 0xF16955B9A50074ED04E1B8A511E35989    3.6480  2.4%  2907 0.0013  0.02 SELECT announcements courses unread_announcements
#    4 0x27543A651E60736B50443886E289E35A    3.3274  2.2%  2905 0.0011  0.01 UPDATE unread_announcements
#    5 0x4A879C906EDA9FDE22E01B64DA82E819    3.0865  2.0%  4243 0.0007  0.00 SELECT registrations
#    6 0x7DFA2D5D9DBC803F79DB97773EC5447B    2.9845  2.0%  1688 0.0018  0.00 INSERT time_zone_transition
#    7 0x2BC75A9C3EB65F1EC570DFB4E3F111C5    2.5699  1.7%  1092 0.0024  0.00 SELECT announcements courses registrations unread_announcements
#    8 0x9E2DA589A20EC24C34E11DDE0FBF5564    2.4024  1.6%  7898 0.0003  0.00 START
#    9 0x9304F82758390EA207A515217B0345EB    1.8926  1.2%     6 0.3154  0.60 INSERT users
#   10 0x34680D6218DB2F97EAE350D366B60E94    1.6845  1.1%   455 0.0037  0.02 INSERT unread_announcements
#   11 0x1F21068CC7652980263E82315944F5FD    1.3489  0.9%  1152 0.0012  0.02 SELECT unread_announcements
#   12 0xBCFA83825ED2F4DBAB748E30997E1B95    1.2326  0.8%  1333 0.0009  0.00 INSERT UPDATE registrations
#   13 0x201AB42EC253BFB0F6BC1153F6B93083    1.2303  0.8%  1824 0.0007  0.00 SELECT courses
#   14 0x6C0FF098D63D248BBF3D40588EFB42EF    1.0186  0.7%   793 0.0013  0.00 UPDATE submissions users
#   15 0x93B0490C6027E6D67E3D4DA357148667    0.9706  0.6%  1789 0.0005  0.00 INSERT time_zone_transition_type
#   16 0xAE994A27799DADD410E848351C4DFF53    0.9589  0.6%   806 0.0012  0.00 SELECT classes submissions
#   17 0x63C05502F034B2DB20500EA0606444D4    0.9210  0.6%  1223 0.0008  0.00 SELECT classes
#   18 0xD22122C68747429FE1B81D07D7E3DD52    0.9129  0.6%  1273 0.0007  0.00 SELECT courses
#   19 0x396201721CD58410E070DA9421CA8C8D    0.9007  0.6%  1298 0.0007  0.00 SELECT users
#   20 0x98DD1BA507AB79261637698A8004A814    0.8547  0.6%   790 0.0011  0.00 INSERT UPDATE submissions
# MISC 0xMISC                               12.3605  8.1% 15121 0.0008   0.0 <79 ITEMS>
```

`INSERT unread_announcements`は解消されたがやはり`COMMIT`がすごい。
Transaction処理におかしなところがないのかを改めて確認しようと思う。
Query数の多い`SELECT announcements cources unread_announcements`,`SELECT registrations`を呼んでるあたりを確認する。
よって、`GetAnnouncementDetail`の実装の悪いところを見てみる

### 見つけたポイント

- 無駄に履修登録の確認をしている処理を廃止
- 既読済なのに毎回既読済にするUPDATEやCOMMITを廃止(早期return)

上記を改修する。

## 無駄にCOMMITしない。未読お知らせのUPDATEを無駄に実施しない(SCORE: 3124)

### Slow Query 確認

改修後のSlowQuery

```
# Profile
# Rank Query ID                            Response time  Calls R/Call V/M
# ==== =================================== ============== ===== ====== ===
#    1 0xFFFCA4D67EA0A788813031B8BBC3B329  107.5313 67.2%  8083 0.0133  0.01 COMMIT
#    2 0xDAFB520F3EA62E0FD8FCB26DC5D1E62F    4.4859  2.8%   243 0.0185  0.00 INSERT courses
#    3 0x7DFA2D5D9DBC803F79DB97773EC5447B    4.3811  2.7%  1688 0.0026  0.01 INSERT time_zone_transition
#    4 0xF16955B9A50074ED04E1B8A511E35989    4.0859  2.6%  2998 0.0014  0.01 SELECT announcements courses unread_announcements
#    5 0x27543A651E60736B50443886E289E35A    3.6054  2.3%  2852 0.0013  0.00 UPDATE unread_announcements
#    6 0x9E2DA589A20EC24C34E11DDE0FBF5564    2.6932  1.7%  8264 0.0003  0.00 START
#    7 0x2BC75A9C3EB65F1EC570DFB4E3F111C5    2.5980  1.6%  1075 0.0024  0.00 SELECT announcements courses registrations unread_announcements
#    8 0x34680D6218DB2F97EAE350D366B60E94    1.8745  1.2%   452 0.0041  0.02 INSERT unread_announcements
#    9 0x201AB42EC253BFB0F6BC1153F6B93083    1.4473  0.9%  1899 0.0008  0.00 SELECT courses
#   10 0x6C0FF098D63D248BBF3D40588EFB42EF    1.3552  0.8%   928 0.0015  0.00 UPDATE submissions users
#   11 0xBCFA83825ED2F4DBAB748E30997E1B95    1.3201  0.8%  1386 0.0010  0.00 INSERT UPDATE registrations
#   12 0x93B0490C6027E6D67E3D4DA357148667    1.3133  0.8%  1789 0.0007  0.00 INSERT time_zone_transition_type
#   13 0x1F21068CC7652980263E82315944F5FD    1.3100  0.8%  1135 0.0012  0.01 SELECT unread_announcements
#   14 0x396201721CD58410E070DA9421CA8C8D    1.2298  0.8%  1680 0.0007  0.00 SELECT users
#   15 0x63C05502F034B2DB20500EA0606444D4    1.2252  0.8%  1349 0.0009  0.01 SELECT classes
#   16 0xD22122C68747429FE1B81D07D7E3DD52    1.2049  0.8%  1401 0.0009  0.00 SELECT courses
#   17 0xAE994A27799DADD410E848351C4DFF53    1.1662  0.7%   941 0.0012  0.00 SELECT classes submissions
#   18 0x9304F82758390EA207A515217B0345EB    1.1141  0.7%     6 0.1857  0.30 INSERT users
#   19 0x98DD1BA507AB79261637698A8004A814    1.1008  0.7%   929 0.0012  0.00 INSERT UPDATE submissions
#   20 0x4A879C906EDA9FDE22E01B64DA82E819    0.9458  0.6%  1408 0.0007  0.00 SELECT registrations
# MISC 0xMISC                               13.9281  8.7% 15484 0.0009   0.0 <79 ITEMS>
```

Commitは仕方ない？INSERT courcesから挑戦していく。

## 外部キー制約をやめて、代わりにINDEXを残す(SCORE: 3283)

### Slow Query 確認

```
# Profile
# Rank Query ID                            Response time  Calls R/Call V/M
# ==== =================================== ============== ===== ====== ===
#    1 0xFFFCA4D67EA0A788813031B8BBC3B329  112.8640 71.7%  8194 0.0138  0.01 COMMIT
#    2 0x27543A651E60736B50443886E289E35A    4.9076  3.1%  2967 0.0017  0.03 UPDATE unread_announcements
#    3 0xDAFB520F3EA62E0FD8FCB26DC5D1E62F    4.0318  2.6%   209 0.0193  0.00 INSERT courses
#    4 0xF16955B9A50074ED04E1B8A511E35989    3.7652  2.4%  3136 0.0012  0.00 SELECT announcements courses unread_announcements
#    5 0x2BC75A9C3EB65F1EC570DFB4E3F111C5    2.8928  1.8%  1137 0.0025  0.00 SELECT announcements courses registrations unread_announcements
#    6 0x9E2DA589A20EC24C34E11DDE0FBF5564    2.7497  1.7%  8412 0.0003  0.00 START
```

Slow Queryの調査からは劇的な改善点が見つからない。
ぐっちょくに次はコード読んで探してみる。

## Count(*)でLIMIT 1 としていい場所はLIMIT 1とする(SCORE: 3346)

