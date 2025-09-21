vcl 4.0;

include "circular1.vcl";

backend circular2_backend {
    .host = "circular2.example.com";
    .port = "8080";
}

sub circular2_sub {
    set req.http.X-Circular = "detected";
}