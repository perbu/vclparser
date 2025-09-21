vcl 4.1;

// Shared subroutines

sub normalize_headers {
    // Remove unwanted headers
    unset req.http.User-Agent;
    unset req.http.X-Forwarded-For;

    // Set normalized headers
    set req.http.X-Normalized = "true";
}

sub process_backend_response {
    // Set cache control headers
    if (beresp.status == 200) {
        set beresp.ttl = 1h;
        set beresp.http.Cache-Control = "public, max-age=3600";
    }

    // Add processing timestamp
    set beresp.http.X-Processed-At = now;
}

sub error_page {
    set resp.http.Content-Type = "text/html";
    set resp.status = 500;
}