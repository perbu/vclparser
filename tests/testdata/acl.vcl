vcl 4.0;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

acl trusted {
    "localhost";
}


sub vcl_recv {
    # Check if the client's IP address matches any entry in the 'trusted_clients' ACL.
    if (client.ip ~ trusted) {
        set req.http.trusted = "true";
    } else {
        set req.http.trusted = "false";
    }

}
