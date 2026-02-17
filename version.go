package main

// Version is set at build time via -ldflags "-X main.Version=x.y.z"
// If not set, defaults to "dev"
var Version = "dev"

// GitCommit is set at build time via -ldflags "-X main.GitCommit=abc123"
var GitCommit = ""
