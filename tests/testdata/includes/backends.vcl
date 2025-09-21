vcl 4.1;

// Backend definitions

backend web1 {
    .host = "192.168.1.10";
    .port = "8080";
    .connect_timeout = 5s;
    .first_byte_timeout = 30s;
    .between_bytes_timeout = 5s;
}

backend web2 {
    .host = "192.168.1.11";
    .port = "8080";
    .connect_timeout = 5s;
    .first_byte_timeout = 30s;
    .between_bytes_timeout = 5s;
}

backend public_backend {
    .host = "203.0.113.100";
    .port = "80";
}

// Director for load balancing
import directors;

sub vcl_init {
    new web_cluster = directors.round_robin();
    web_cluster.add_backend(web1);
    web_cluster.add_backend(web2);
}