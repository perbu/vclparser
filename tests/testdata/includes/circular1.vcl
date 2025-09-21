vcl 4.0;

include "circular2.vcl";

backend circular1_backend {
    .host = "circular1.example.com";
    .port = "80";
}

sub circular1_sub {
    set req.http.X-Circular1 = "processed";
}