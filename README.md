<div align="center">
  <a href="https://timur.mobi/webcall"><img src="webroot/webcall-logo.png" alt="WebCall"></a>
</div>

# WebCall WebRTC Telephony Server

Browser based telephony over E2EE P2P with very high audio quality.
WebCall is lightweight and easy to use.

New: WebCall offers device-to-device, bidirectional file transfer. 
If you need to send files to someone quick, I cannot think of a better solution.
And you can use the voice feature to make sure only the right person gets the files.

[https://timur.mobi/webcall](https://timur.mobi/webcall)


# Decentral WebCall

We can build a network of telephony servers, offering completely free and
boundless telephony to everyone. This is possible because WebCall
servers can be operated by anyone. And because, as a user, you can interact with any
other WebCall user, regardless of who is hosting you.
All of this is possible without central coordination and without any server-to-server
communication taking place.

[https://timur.mobi/webcall/info/server](https://timur.mobi/webcall/info/server)


# Installation

A single WebCall server can provide free and secure telephony to 1M users.
A very small (1GB, 1core, shared) server can host up to 50K concurrent users.
Here I describe how you can build WebCall and run it for yourself and your friends:

[https://timur.mobi/webcall/install](https://timur.mobi/webcall/install)


# License

AGPL3.0 - see: [LICENSE](LICENSE)


# 3rd party code (external)

- github.com/lesismal/nbio
- go.etcd.io/bbolt
- github.com/pion/turn
- github.com/mrjones/oauth
- github.com/SherClockHolmes/webpush-go
- gopkg.in/ini
- github.com/webrtcHacks/adapter.js

# 3rd party code (embedded/modified)

- github.com/RapidLoop: skv
- github.com/ChimeraCoder: Twitter
- AppRTC's sdputils.js: prefercodec.js
- mr-wang-from-next-door: GetOutboundIP()
- github.com/mahan: AtomBool

