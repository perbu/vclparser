// Subroutines with call statements

acl internal_ips {
    "127.0.0.1";
    "10.0.0.0"/8;
}

backend web_cluster {
    .host = "cluster.example.com";
    .port = "80";
}

sub normalize_headers {
    unset req.http.User-Agent;
    set req.http.X-Normalized = "true";
    call log_processing;
}

sub validate_request {
    if (req.method !~ "^(GET|HEAD|POST)$") {
        return (synth(405, "Method not allowed"));
    }
    call check_rate_limit;
}

sub process_backend_response {
    if (beresp.status == 200) {
        set beresp.ttl = 1h;
        call add_cache_headers;
    }
}

sub log_processing {
    set req.http.X-Processing-Step = "headers_normalized";
}

sub check_rate_limit {
    set req.http.X-Rate-Check = "passed";
}

sub add_cache_headers {
    set beresp.http.Cache-Control = "public, max-age=3600";
}