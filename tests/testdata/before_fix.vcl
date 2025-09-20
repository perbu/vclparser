vcl 4.1;

sub test_before_fix {
    // This should work (parenthesized)
    return (lookup);
    return (hash);
    return (pass);
}

sub test_naked_returns {
    // These should now work without parentheses
    return lookup;
    return hash;
    return pass;
    return vcl;
}