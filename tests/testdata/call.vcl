vcl 4.1;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

sub normalize_headers {
    // Remove unwanted headers
    unset req.http.User-Agent;
    unset req.http.X-Forwarded-For;

    // Set normalized headers
    set req.http.X-Normalized = "true";
}


sub vcl_recv {
    call normalize_headers;
}
