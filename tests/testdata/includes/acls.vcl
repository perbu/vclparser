vcl 4.1;

// Access Control Lists

acl internal_ips {
    "127.0.0.1";
    "10.0.0.0"/8;
    "192.168.0.0"/16;
    "172.16.0.0"/12;
}

acl admin_ips {
    "192.168.1.100";
    "192.168.1.101";
    "203.0.113.50";
}

acl monitoring_ips {
    "10.1.1.0"/24;
    "monitoring.example.com";
}