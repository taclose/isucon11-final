
# 当日も使いそうなコマンドや設定変更内容をメモ

## コマンドまとめ

```bash
# nginx restart
$ sudo systemctl reload nginx

# nginx シンタックスエラーが起きていない事を確認
$ sudo nginx -t

# logを削除(json形式じゃないlogが解析の邪魔となる）
$ sudo rm /var/log/nginx/access.log

```

## nginx 設定変更

### ログフォーマット変更 nginx.conf
```text
log_format json escape=json '{"time": "$time_iso8601",'
                                '"host": "$remote_addr",'
                                '"port": "$remote_port",'
                                '"method": "$request_method",'
                                '"uri": "$request_uri",'
                                '"status": "$status",'
                                '"body_bytes": "$body_bytes_sent",'
                                '"referer": "$http_referer",'
                                '"ua": "$http_user_agent",'
                                '"request_time": "$request_time",'
                                '"response_time": "$upstream_response_time"}';
access_log /var/log/nginx/access.log json;
```
### nginx log 解析

```text
# unzipコマンドをインストール
$ sudo apt-get install zip

# alpをインストール
$ sudo wget https://github.com/tkuchiki/alp/releases/download/v1.0.16/alp_linux_amd64.zip
$ sudo unzip alp_linux_amd64.zip
$ sudo install alp /usr/local/bin/alp

$ cat /var/log/nginx/access.log | alp json -o count,method,uri,min,avg,max,sum -m "/image/.+,/posts/[0-9]+,/@.+" --sort=sum -r
```

## mysql 設定変更

### slow log 出力

```text
[mysqld]
slow_query_log         = 1
slow_query_log_file    = /var/log/mysql/mysql-slow.log
long_query_time = 0
```

### slow log 解析

```text
# pt-query-digestの実行
$ sudo pt-query-digest  --group-by fingerprint --order-by Query_time:sum /var/log/mysql/mysql-slow.log
```







