## You know for forwarding journald
Example:

journalctl --system -f -o json | journalfwd -host=sysloghost -port=syslog_port