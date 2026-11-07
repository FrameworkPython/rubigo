// Package parsrubika is a high-performance Go library for the Rubika
// user-account API, written from scratch with a focus on speed,
// security and ergonomics.
//
// The user / account client (in `parsrubika/account`) speaks the
// encrypted Rubika API (AES-CBC + RSA-PKCS1v15 signing) and listens
// for live updates over a WebSocket connection. It is built on a
// fast, secure foundation of small single-purpose packages:
//
// - parsrubika/core       : version, author, constants, typed errors
// - parsrubika/security   : AES-CBC, RSA-PKCS1v15, key generation
// - parsrubika/text       : markdown / html → metadata formatter
// - parsrubika/enums      : shared enumerations (parse mode, chat type, ...)
// - parsrubika/storage    : pluggable session storage (sqlite / memory / file / string)
// - parsrubika/events     : dynamic event wrapper used by the account client
// - parsrubika/transport  : HTTP/2, WebSocket, DC discovery, sync.Pool
// - parsrubika/files      : chunked uploader / downloader with progress + thumbnails
// - parsrubika/account    : the user / account client (encrypted API + WebSocket)
// - parsrubika/logger     : optional coloured / JSON / rotating-file logger
//
// Author : Abolfazl Zarei
// Version: 3.8.0
// License: LGPL v3
package parsrubika
