vcl 4.1;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

sub vcl_recv {
    if (req.method == "GET") {
        return(hash);
    }

    if (req.method == "POST") {
        return(pass);
    }

    if (req.method == "PUT") {
        return(pipe);
    }

    if (req.method == "DELETE") {
        return(purge);
    }

    if (req.method == "OPTIONS") {
        return(synth(200, "OK"));
    }

    // Test return statements with different actions
    if (req.url ~ "^/admin") {
        return(fail);
    }

    // Default action
    return(pass);
}

sub vcl_hash {
    if (req.http.host) {
        return(lookup);
    } else {
        return(fail);
    }
}

sub vcl_hit {
}

sub vcl_miss {
    if (req.http.no-cache) {
        return(pass);
    } else {
        return(fetch);
    }
}

sub vcl_backend_fetch {
    if (bereq.method == "GET") {
        return(fetch);
    } else {
        return(abandon);
    }
}

sub vcl_backend_response {
    if (beresp.status >= 500) {
        return(retry);
    } else {
        return(deliver);
    }
}

sub vcl_backend_error {
    if (beresp.status == 503) {
        return(retry);
    } else {
        return(deliver);
    }
}

sub vcl_deliver {
    if (resp.status == 200) {
        return(deliver);
    } else {
        return(restart);
    }
}

sub vcl_synth {
    if (resp.status == 404) {
        return(deliver);
    } else {
        return(restart);
    }
}
