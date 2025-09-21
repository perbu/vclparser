vcl 4.1;

include "call_subroutines.vcl";

sub vcl_recv {
    if (client.ip ~ internal_ips) {
        set req.backend_hint = web_cluster;
    }

    call normalize_headers;
    call validate_request;
}

sub vcl_backend_response {
    call process_backend_response;
}