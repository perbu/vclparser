vcl 4.0;

include "nested_level2.vcl";

backend level1_backend {
    .host = "level1.example.com";
    .port = "80";
}

sub level1_sub {
    set req.http.X-Level1 = "processed";
    set req.backend_hint = level1_backend;
}