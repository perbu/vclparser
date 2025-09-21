vcl 4.0;

include "nested_level1.vcl";

sub vcl_recv {
    set req.backend_hint = level1_backend;
}