vcl 4.1;

import std;
import directors;

backend web1 {
    .host = "192.168.1.10";
    .port = "80";
    .probe = {
        .url = "/health";
        .interval = 5s;
        .timeout = 2s;
        .window = 5;
        .threshold = 3;
    };
}

backend web2 {
    .host = "192.168.1.11";
    .port = "80";
    .probe = healthcheck;
}

probe healthcheck {
    .url = "/health";
    .interval = 30s;
    .timeout = 5s;
    .window = 5;
    .threshold = 3;
}

acl purge {
    "localhost";
    "127.0.0.1";
    "::1";
    !"192.168.1.100";
}

sub vcl_init {
    new cluster = directors.round_robin();
    cluster.add_backend(web1);
    cluster.add_backend(web2);
}

sub vcl_recv {
    set req.backend_hint = cluster.backend();

    if (req.method == "PURGE") {
        if (!client.ip ~ purge) {
            return (synth(405, "Not allowed"));
        }
        return (purge);
    }

    if (req.method != "GET" &&
        req.method != "HEAD" &&
        req.method != "PUT" &&
        req.method != "POST" &&
        req.method != "TRACE" &&
        req.method != "OPTIONS" &&
        req.method != "DELETE") {
        return (pipe);
    }

    if (req.method != "GET" && req.method != "HEAD") {
        return (pass);
    }

    if (req.http.Authorization || req.http.Cookie) {
        return (pass);
    }

    return (hash);
}

sub vcl_backend_response {
    if (beresp.ttl <= 0s ||
        beresp.http.Set-Cookie ||
        beresp.http.Vary == "*") {
        set beresp.ttl = 120s;
        set beresp.uncacheable = true;
        return (deliver);
    }

    set beresp.ttl = 1h;
    return (deliver);
}

sub vcl_deliver {
    if (obj.hits > 0) {
        set resp.http.X-Cache = "HIT";
    } else {
        set resp.http.X-Cache = "MISS";
    }
    set resp.http.X-Cache-Hits = obj.hits;
}

sub vcl_synth {
    if (resp.status == 720) {
        set resp.status = 301;
        set resp.http.Location = "https://example.com" + req.url;
        return (deliver);
    }
}