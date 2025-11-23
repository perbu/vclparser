// VCL Configuration with Comments Example
vcl 4.1;

// Import the standard VMOD
import std;

/*
 * Backend Configuration
 * This section defines the backend servers
 */

// Primary backend server
backend web1 {
    .host = "192.168.1.10";
    .port = "8080";
}

// Backup backend server
backend web2 {
    .host = "192.168.1.11";
    .port = "8080";
}

/*
 * Request Handler
 */

// Main request processing subroutine
sub vcl_recv {
    // Normalize the host header
    set req.http.Host = regsub(req.http.Host, ":[0-9]+", "");

    // Handle different request methods
    if (req.method == "GET" || req.method == "HEAD") {
        // Cache GET and HEAD requests
        return(hash); // Look up in cache
    }

    // Don't cache POST requests
    if (req.method == "POST") {
        return(pass);
    }

    // Pass everything else through
    return(pass);
}

// Backend response handler
sub vcl_backend_response {
    // Set cache TTL based on status code
    if (beresp.status == 200) {
        set beresp.ttl = 1h; // Cache for 1 hour
    }

    // Don't cache errors
    if (beresp.status >= 500) {
        return(abandon);
    }

    return(deliver);
}
