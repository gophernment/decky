package web

import "embed"

// Assets is the embedded filesystem containing static assets and templates.
//
//go:embed templates/* static/*
var Assets embed.FS
