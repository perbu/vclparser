vcl 4.1;

// Test various naked return statement syntaxes
// This file tests that the parser can handle return statements without parentheses

sub vcl_recv {
    if (req.method == "GET") {
        return hash;
    }

    if (req.method == "POST") {
        return pass;
    }

    if (req.method == "PUT") {
        return pipe;
    }

    if (req.method == "DELETE") {
        return purge;
    }

    if (req.method == "OPTIONS") {
        return synth(200, "OK");
    }

    // Test naked return with different actions
    if (req.url ~ "^/admin") {
        return fail;
    }

    // Default action
    return lookup;
}

sub vcl_hash {
    if (req.http.host) {
        return lookup;
    } else {
        return fail;
    }
}

sub vcl_hit {
    if (req.method == "GET") {
        return deliver;
    } else {
        return miss;
    }
}

sub vcl_miss {
    if (req.http.no-cache) {
        return pass;
    } else {
        return fetch;
    }
}

sub vcl_backend_fetch {
    if (bereq.method == "GET") {
        return fetch;
    } else {
        return abandon;
    }
}

sub vcl_backend_response {
    if (beresp.status >= 500) {
        return retry;
    } else {
        return deliver;
    }
}

sub vcl_backend_error {
    if (beresp.status == 503) {
        return retry;
    } else {
        return deliver;
    }
}

sub vcl_deliver {
    if (resp.status == 200) {
        return deliver;
    } else {
        return restart;
    }
}

sub vcl_synth {
    if (resp.status == 404) {
        return deliver;
    } else {
        return restart;
    }
}

// Test mixed syntax - naked and parenthesized
sub test_mixed {
    if (req.method == "GET") {
        return hash;  // naked
    } else {
        return (pass);  // parenthesized
    }
}