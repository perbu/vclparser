vcl 4.1;

include "backends.vcl";
include "subroutines.vcl";
include "acls.vcl";

sub vcl_recv {
    if (client.ip ~ internal_ips) {
        set req.backend_hint = web_cluster;
    } else {
        set req.backend_hint = public_backend;
    }

    set req.http.X-Normalized = "true";
}

sub vcl_backend_response {
    set beresp.ttl = 1h;
}