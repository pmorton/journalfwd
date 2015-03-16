## You know for forwarding journald

Opens Stdout of journalctl for system and dmesg and forwards them to a remote syslog server

```bash
journalfwd -host=sysloghost -port=syslog_port
```

Example Systemd Unit
```bash
[Unit]
Description=Journal Forwarding

[Service]
ExecStartPre=/usr/bin/curl -L https://bintray.com/artifact/download/pmorton/journalfwd/journalfwd_linux_amd64 -o /var/run/journalfwd
ExecStartPre=/usr/bin/chmod +x /var/run//journalfwd
ExecStart=/var/run/journalfwd -host logs2.papertrailapp.com -port <your port>  -color -useTcp
```