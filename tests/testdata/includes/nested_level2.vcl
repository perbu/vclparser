vcl 4.0;

backend level2_backend {
    .host = "level2.example.com";
    .port = "8080";
}

acl level2_acl {
    "203.0.113.0"/24;
}

sub level2_sub {
    set req.http.X-Level2 = "processed";
}