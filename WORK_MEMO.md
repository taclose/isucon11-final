## 改修前
19:09:52.810301 raw score (22321) breakdown:

## slow query 調査

```text
# Profile
# Rank Query ID                     Response time  Calls  R/Call V/M   Ite
# ==== ============================ ============== ====== ====== ===== ===
#    1 0xFFFCA4D67EA0A788813031B... 477.3832 88.5%  27869 0.0171  0.10 COMMIT
#    2 0xDA556F9115773A1A99AA016...  12.0965  2.2% 127088 0.0001  0.00 ADMIN PREPARE
#    3 0xDAFB520F3EA62E0FD8FCB26...   9.1178  1.7%    333 0.0274  0.01 INSERT courses
#    4 0x98DD1BA507AB79261637698...   2.8628  0.5%   6632 0.0004  0.01 INSERT UPDATE submissions
#    5 0xA9A13C7DE4DE6522B8635F3...   2.8491  0.5%   1955 0.0015  0.00 SELECT users registrations courses classes submissions
#    6 0x2BC75A9C3EB65F1EC570DFB...   2.4405  0.5%   3223 0.0008  0.00 SELECT announcements courses registrations unread_announcements
#    7 0x8082220C07E2A565D306665...   2.2807  0.4%     56 0.0407  0.01 SELECT users registrations courses registrations courses classes submissions
#    8 0xAE994A27799DADD410E8483...   2.1674  0.4%   6690 0.0003  0.01 SELECT classes submissions
#    9 0x34680D6218DB2F97EAE350D...   2.0847  0.4%   9211 0.0002  0.01 INSERT unread_announcements
# MISC 0xMISC                        26.1363  4.8% 263184 0.0001   0.0 <92 ITEMS>
```
